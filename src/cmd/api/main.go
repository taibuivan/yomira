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
	must(log, err, "load configuration")

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

	// Root context for startup. A 30s deadline prevents the app from hanging on misconfiguration.
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()

	// # 3. PostgreSQL
	pool, err := pgstore.NewPool(startupCtx, cfg.DatabaseURL, log)
	must(log, err, "connect to postgres")

	// Ensure pool is closed during orderly shutdown
	defer func() {
		log.Info("closing postgres pool")
		pool.Close()
	}()

	// # 4. Redis
	rdb, err := redisstore.NewClient(startupCtx, cfg.RedisURL, log)
	must(log, err, "connect to redis")

	// Ensure client is closed during orderly shutdown
	defer func() {
		log.Info("closing redis client")
		if cerr := rdb.Close(); cerr != nil {
			log.Error("redis close error", slog.Any("error", cerr))
		}
	}()

	// # 5. Migrations
	// Idempotent migrations ensure schema is always up to date.
	must(log, migration.RunUp(cfg.DatabaseURL, cfg.MigrationPath, log), "run migrations")

	// # 6. Platform Services
	jwtSvc, err := sec.NewTokenService(cfg.JWTPrivKeyPath, cfg.JWTPubKeyPath, constants.AuthIssuer)
	must(log, err, "initialize jwt service")

	// # 7. Health Wiring
	liveness, readiness := api.NewHealthHandlers(api.HealthDependencies{
		CheckDatabase: func() error {
			return pgstore.Ping(context.Background(), pool)
		},
		CheckCache: func() error {
			return redisstore.Ping(context.Background(), rdb)
		},
	}, log)

	// # 8. Domain Domain (Auth)
	userRepository := auth.NewUserRepository(pool)
	sessionRepository := auth.NewSessionRepository(pool)
	resetRepository := auth.NewResetTokenRepository(rdb)
	verificationRepository := auth.NewVerificationTokenRepository(rdb)

	authService := auth.NewService(
		userRepository,
		sessionRepository,
		resetRepository,
		verificationRepository,
		jwtSvc,
	)
	authHandler := auth.NewHandler(authService)

	// # 9. Domain Domain (Comic)
	comicRepo := comic.NewComicRepository(pool)
	chapterRepo := comic.NewChapterRepository(pool)

	comicService := comic.NewService(comicRepo, chapterRepo)
	comicHandler := comic.NewHandler(comicService)

	// # 9.1 Domain Domain (Reference)
	refRepo := reference.NewPostgresRepository(pool)
	refService := reference.NewService(refRepo)
	refHandler := reference.NewHandler(refService)

	// # 9.2 Domain Domain (Group)
	groupRepo := group.NewPostgresRepository(pool)
	groupService := group.NewService(groupRepo)
	groupHandler := group.NewHandler(groupService)

	// # 10. API Server Assembly
	handlers := api.Handlers{
		Liveness:  liveness,
		Readiness: readiness,
		Auth:      authHandler,
		Comic:     comicHandler,
		Reference: refHandler,
		Group:     groupHandler,
	}

	// Create a background context that we can cancel to stop background workers
	appCtx, appCancel := context.WithCancel(context.Background())

	// Ensure background workers are stopped during orderly shutdown
	defer appCancel()

	server := api.NewServer(appCtx, cfg, log, jwtSvc, handlers)

	// # 10. Lifecycle Management (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	serverErr := make(chan error, 1)
	go func() {
		// Start the HTTP server listener in a non-blocking goroutine
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Block until a signal is received or the server crashes
	select {
	case sig := <-quit:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))

		// Explicitly stop background tasks now
		appCancel()
	case err := <-serverErr:
		log.Error("server startup error", slog.Any("error", err))
	}

	// Give in-flight requests a window to finish before forced exit
	shutdownTimeout := constants.ShutdownTimeout
	log.Info("shutting down server", slog.Duration("timeout", shutdownTimeout))

	if err := server.Shutdown(shutdownTimeout); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("server stopped cleanly")
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
