// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package postgres provides a high-performance PostgreSQL driver and connection pool.

It specializes in managing 'pgxpool' instances, ensuring that database connections
are recycled efficiently and timeouts are enforced at the driver level.

Architecture:

  - Pool: Thread-safe connection pooling with automatic health checks (Ping).
  - Tuning: Configures MaxConns, MinConns, and MaxConnIdleTime for scalability.
  - Safety: Integrates context deadlines to prevent runaway queries.

This package acts as the bridge between the domain repositories and the physical
storage layer.
*/
package postgres

import (
	stdctx "context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taibuivan/yomira/internal/platform/constants"
)

// # Pool Configuration (Tuning)

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

// # Lifecycle Management

// NewPool creates and validates a new PostgreSQL connection pool.
func NewPool(context stdctx.Context, dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {

	// Step 1: Parse the DSN string
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: invalid DSN: %w", err)
	}

	// Step 2: Apply pool tuning parameters
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = maxConnLifetime
	poolConfig.MaxConnIdleTime = maxConnIdleTime
	poolConfig.HealthCheckPeriod = healthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = connectTimeout

	// AfterConnect is called each time a new physical connection is established.
	// We use it to set a per-connection statement timeout for safety.
	poolConfig.AfterConnect = func(context stdctx.Context, connection *pgx.Conn) error {

		timeoutQuery := fmt.Sprintf("SET statement_timeout = '%ds'", int(constants.GlobalRequestTimeout.Seconds()))
		_, err := connection.Exec(context, timeoutQuery)
		return err
	}

	// Step 3: Establish the pool
	connectCtx, cancel := stdctx.WithTimeout(context, connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to create pool: %w", err)
	}

	// Step 4: Validate that we can actually reach the database
	if err := Ping(context, pool); err != nil {
		pool.Close()
		return nil, err
	}

	// Step 5: Log pool statistics on startup
	stats := pool.Stat()
	logger.Info("postgres pool connected",
		slog.Int("max_conns", int(stats.MaxConns())),
		slog.Int("total_conns", int(stats.TotalConns())),
	)

	return pool, nil
}

// # Health Checks

// Ping verifies that the PostgreSQL connection pool is healthy.
func Ping(context stdctx.Context, pool *pgxpool.Pool) error {

	// Execute a lightweight ping with a strict timeout
	pingCtx, cancel := stdctx.WithTimeout(context, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("postgres: ping failed: %w", err)
	}

	return nil
}
