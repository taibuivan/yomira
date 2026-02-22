// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package redis provides a managed Redis client for the Yomira application.
//
// # Architecture
//
// This package handles fast, volatile storage operations (Caching, Rate Limiting).
// It abstracts away the connection pool details from the Application layer.
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Opiniated default timeouts for Redis operations.
const (
	dialTimeout  = 3 * time.Second
	readTimeout  = 2 * time.Second
	writeTimeout = 2 * time.Second
	pingTimeout  = 2 * time.Second
)

// NewClient parses a Redis URL and returns a ready-to-use client.
//
// # Parameters
//   - ctx: Context for the initial ping.
//   - redisURL: Redis connection URL.
//   - logger: Structured logger for connection events.
func NewClient(ctx context.Context, redisURL string, logger *slog.Logger) (*redis.Client, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid URL: %w", err)
	}

	// Pool configuration Tuning
	options.PoolSize = 10
	options.MinIdleConns = 2
	options.MaxIdleConns = 5

	options.DialTimeout = dialTimeout
	options.ReadTimeout = readTimeout
	options.WriteTimeout = writeTimeout

	client := redis.NewClient(options)

	// Validate connectivity immediately at startup.
	if err := Ping(ctx, client); err != nil {
		_ = client.Close()
		return nil, err
	}

	logger.Info("redis client connected",
		slog.String("addr", options.Addr),
		slog.Int("pool_size", options.PoolSize),
	)

	return client, nil
}

// Ping verifies that the Redis client is healthy.
func Ping(ctx context.Context, client *redis.Client) error {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("redis: ping failed: %w", err)
	}

	return nil
}
