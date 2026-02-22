// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Command api is the entry point for the Yomira HTTP API server.
//
// # Startup Sequence
//
//  1. Initialize structured logger.
//  2. Load configuration from environment variables.
//  3. Connect to PostgreSQL (pgxpool).
//  4. Connect to Redis.
//  5. Run database migrations (idempotent).
//  6. Wire HTTP handlers.
//  7. Start HTTP server with graceful shutdown.
//
// No business logic lives here. All wiring is explicit constructor injection.
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

	"github.com/buivan/yomira/internal/api"
	"github.com/buivan/yomira/internal/auth"
	"github.com/buivan/yomira/internal/platform/config"
	"github.com/buivan/yomira/internal/platform/constants"
	"github.com/buivan/yomira/internal/platform/migration"
	pgstore "github.com/buivan/yomira/internal/platform/postgres"
	redisstore "github.com/buivan/yomira/internal/platform/redis"
	"github.com/buivan/yomira/internal/platform/sec"
)

func main() {
	// ── 1. Logger ──────────────────────────────────────────────────────────
	// Initialize first so that subsequent startup errors are structured JSON.
	rawLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Add global context to all log entries.
	log := rawLog.With(slog.String("app", "yomira"))
	slog.SetDefault(log)

	log.Info("[Yomira] service_initializing")

	// ── 2. Configuration ──────────────────────────────────────────────────
	cfg, err := config.Load()
	must(log, err, "load configuration")

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

	// Root context for startup. Use a 30s deadline so misconfiguration is
	// caught quickly rather than hanging indefinitely.
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()

	// ── 3. PostgreSQL ─────────────────────────────────────────────────────
	pool, err := pgstore.NewPool(startupCtx, cfg.DatabaseURL, log)
	must(log, err, "connect to postgres")
	defer func() {
		log.Info("closing postgres pool")
		pool.Close()
	}()

	// ── 4. Redis ──────────────────────────────────────────────────────────
	rdb, err := redisstore.NewClient(startupCtx, cfg.RedisURL, log)
	must(log, err, "connect to redis")
	defer func() {
		log.Info("closing redis client")
		if cerr := rdb.Close(); cerr != nil {
			log.Error("redis close error", slog.Any("error", cerr))
		}
	}()

	// ── 5. Migrations ─────────────────────────────────────────────────────
	must(log, migration.RunUp(cfg.DatabaseURL, cfg.MigrationPath, log), "run migrations")

	// ── 6. Auth Service ───────────────────────────────────────────────────
	jwtSvc, err := sec.NewTokenService(cfg.JWTPrivKeyPath, cfg.JWTPubKeyPath, constants.AuthIssuer)
	must(log, err, "initialize jwt service")
	// ── 7. Health handlers (wired with real dependency checkers) ──────────
	liveness, readiness := api.NewHealthHandlers(api.HealthDependencies{
		CheckDatabase: func() error {
			return pgstore.Ping(context.Background(), pool)
		},
		CheckCache: func() error {
			return redisstore.Ping(context.Background(), rdb)
		},
	}, log)

	// ── 8. Domain Wiring ──────────────────────────────────────────────────
	userRepository := auth.NewUserRepository(pool)
	sessionRepository := auth.NewSessionRepository(pool)
	authService := auth.NewService(userRepository, sessionRepository, jwtSvc)
	authHandler := auth.NewHandler(authService)

	// ── 9. HTTP Server ────────────────────────────────────────────────────
	handlers := api.Handlers{
		Liveness:  liveness,
		Readiness: readiness,
		Auth:      authHandler,
	}

	server := api.NewServer(cfg, log, jwtSvc, handlers)

	// ── 8. Graceful Shutdown ──────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Block until OS signal or server error.
	select {
	case sig := <-quit:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-serverErr:
		log.Error("server startup error", slog.Any("error", err))
	}

	// Give in-flight requests enough time to complete.
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
