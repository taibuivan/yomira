// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package postgres provides a managed PostgreSQL connection pool and
// repository factory for the Yomira application.
//
// # Architecture
//
// This package is part of the Infrastructure layer. It manages the physical
// database connections (pgxpool) and provides concrete implementations
// for the interfaces defined in the domain layer.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buivan/yomira/internal/platform/constants"
)

// Opinionated pool settings for the Yomira workload.
const (
	// maxConns is the maximum number of connections in the pool.
	maxConns = 25
	// minConns keeps a warm set of connections to avoid cold-start latency.
	minConns = 5
	// maxConnLifetime ensures connections are periodically recycled.
	maxConnLifetime = 60 * time.Minute
	// maxConnIdleTime closes connections that have been idle too long.
	maxConnIdleTime = 10 * time.Minute
	// healthCheckPeriod is the frequency of background connection health checks.
	healthCheckPeriod = 1 * time.Minute
	// connectTimeout is the maximum time allowed to establish a new connection.
	connectTimeout = 5 * time.Second
	// pingTimeout is the maximum duration for a health check ping.
	pingTimeout = 2 * time.Second
)

// NewPool creates and validates a new PostgreSQL connection pool.
//
// # Parameters
//   - ctx: Context for the initial connection attempt.
//   - dsn: A libpq-compatible connection string or postgres:// URL.
//   - logger: Structured logger for pool-level events.
func NewPool(ctx context.Context, dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: invalid DSN: %w", err)
	}

	// Apply pool tuning parameters.
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = maxConnLifetime
	poolConfig.MaxConnIdleTime = maxConnIdleTime
	poolConfig.HealthCheckPeriod = healthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = connectTimeout

	// AfterConnect is called each time a new physical connection is established.
	poolConfig.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		// Set a per-connection statement timeout to avoid runaway queries.
		// Use GlobalRequestTimeout as the baseline for safety.
		timeoutQuery := fmt.Sprintf("SET statement_timeout = '%ds'", int(constants.GlobalRequestTimeout.Seconds()))
		_, err := connection.Exec(ctx, timeoutQuery)
		return err
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to create pool: %w", err)
	}

	// Validate that we can actually reach the database.
	if err := Ping(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	stats := pool.Stat()
	logger.Info("postgres pool connected",
		slog.Int("max_conns", int(stats.MaxConns())),
		slog.Int("total_conns", int(stats.TotalConns())),
	)

	return pool, nil
}

// Ping verifies that the PostgreSQL connection pool is healthy.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("postgres: ping failed: %w", err)
	}

	return nil
}
