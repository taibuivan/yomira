// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Api is the entry point for the Yomira HTTP API server.

The server provides a high-performance, secure backend for the Yomira comic platform.
It handles everything from user identity and session management to comic metadata
and chapter delivery.

Usage:

	go run cmd/api/main.go [flags]

The flags/environment variables are:

	SERVER_PORT     Port to listen on (default: 8080)
	ENVIRONMENT     deployment environment (development, production)
	DATABASE_URL    Postgres connection string (required)
	REDIS_URL       Redis connection string (required)

Startup Sequence:

 1. Logger: Initialize structured JSON logging (slog).
 2. Config: Load and validate environment variables.
 3. Storage: Establish connections to Postgres and Redis.
 4. Migration: Run idempotent schema updates.
 5. Wiring: Inject dependencies into domain services/handlers.
 6. Server: Bind HTTP listener and handle graceful shutdown.

No business logic lives here. This file is strictly for orchestration and wiring.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taibuivan/yomira/internal/api"
	"github.com/taibuivan/yomira/internal/core/comic"
	"github.com/taibuivan/yomira/internal/core/group"
	"github.com/taibuivan/yomira/internal/core/reference"
	"github.com/taibuivan/yomira/internal/platform/config"
	"github.com/taibuivan/yomira/internal/platform/constants"
	"github.com/taibuivan/yomira/internal/platform/migration"
	pgstore "github.com/taibuivan/yomira/internal/platform/postgres"
	redisstore "github.com/taibuivan/yomira/internal/platform/redis"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/internal/users/auth"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application_startup_failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// # 1. Logger
	// Initialize first so that subsequent startup errors are structured JSON.
	rawLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Add global context to all log entries for trace correlation
	log := rawLog.With(slog.String("app", "yomira"))
	slog.SetDefault(log)

	log.Info("[Yomira] service_initializing")

	// # 2. Configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Adjust log level if debug mode is explicitly enabled
	if cfg.Debug {
		debugLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		log = debugLog.With(slog.String("app", "yomira"))
		slog.SetDefault(log)
		log.Debug("debug_logging_enabled")
	}

	log.Info("configuration_loaded",
		slog.String("environment", cfg.Environment),
		slog.String("port", cfg.ServerPort),
	)

	// Root context for startup. A 30s deadline prevents the app from hanging.
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()

	// # 3. PostgreSQL
	pool, err := pgstore.NewPool(startupCtx, cfg.DatabaseURL, log)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer func() {
		log.Info("closing postgres pool")
		pool.Close()
	}()

	// # 4. Redis
	rdb, err := redisstore.NewClient(startupCtx, cfg.RedisURL, log)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer func() {
		log.Info("closing redis client")
		if cerr := rdb.Close(); cerr != nil {
			log.Error("redis close error", slog.Any("error", cerr))
		}
	}()

	// # 5. Migrations
	if err := migration.RunUp(cfg.DatabaseURL, cfg.MigrationPath, log); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// # 6. Platform Services
	jwtSvc, err := sec.NewTokenService(cfg.JWTPrivKeyPath, cfg.JWTPubKeyPath, constants.AuthIssuer)
	if err != nil {
		return fmt.Errorf("initialize jwt service: %w", err)
	}

	// # 7. Health Wiring
	liveness, readiness := api.NewHealthHandlers(api.HealthDependencies{
		CheckDatabase: func() error {
			return pgstore.Ping(context.Background(), pool)
		},
		CheckCache: func() error {
			return redisstore.Ping(context.Background(), rdb)
		},
	}, log)

	// # 8. Domain Wiring (Shared Repositories)
	userRepo := auth.NewUserRepository(pool)
	sessionRepo := auth.NewSessionRepository(pool)
	resetRepo := auth.NewResetTokenRepository(rdb)
	verifyRepo := auth.NewVerificationTokenRepository(rdb)

	// # 9. Auth Service & Handler
	authSvc := auth.NewService(userRepo, sessionRepo, resetRepo, verifyRepo, jwtSvc)
	authHdl := auth.NewHandler(authSvc)

	// # 10. Comic Service & Handler
	comicRepo := comic.NewComicRepository(pool)
	chapterRepo := comic.NewChapterRepository(pool)
	comicSvc := comic.NewService(comicRepo, chapterRepo)
	comicHdl := comic.NewHandler(comicSvc)

	// # 11. Reference & Group
	refSvc := reference.NewService(reference.NewPostgresRepository(pool))
	refHdl := reference.NewHandler(refSvc)
	groupSvc := group.NewService(group.NewPostgresRepository(pool))
	groupHdl := group.NewHandler(groupSvc)

	// # 12. API Assembly
	handlers := api.Handlers{
		Liveness:  liveness,
		Readiness: readiness,
		Auth:      authHdl,
		Comic:     comicHdl,
		Reference: refHdl,
		Group:     groupHdl,
	}

	// Create a background context for the whole application lifecycle
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	server := api.NewServer(appCtx, cfg, log, jwtSvc, handlers)

	// # 13. Lifecycle Handling
	shutdownErr := make(chan error, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownErr <- fmt.Errorf("http_server_crash: %w", err)
		}
	}()

	log.Info("yomira_api_running", slog.String("port", cfg.ServerPort))

	// Block until signal or error
	select {
	case sig := <-quit:
		log.Info("shutdown_signal_received", slog.String("signal", sig.String()))
	case err := <-shutdownErr:
		return err
	}

	// Start Graceful Shutdown Sequence
	appCancel() // Signal background workers to stop

	log.Info("shutting_down_api_server", slog.Duration("timeout", constants.ShutdownTimeout))
	if err := server.Shutdown(constants.ShutdownTimeout); err != nil {
		return fmt.Errorf("server_shutdown_failed: %w", err)
	}

	log.Info("graceful_shutdown_complete")
	return nil
}

// must logs a structured fatal error and terminates the process if err is non-nil.
//
// It is intentionally limited to startup wiring. After startup, all errors
// must be returned and handled explicitly (never panic).
func must(log *slog.Logger, err error, context string) {
	if err != nil {
		log.Error("startup failure",
			slog.String("context", context),
			slog.Any("error", err),
		)
		os.Exit(1)
	}
}
