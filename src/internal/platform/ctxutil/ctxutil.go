// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package ctxutil provides helpers for interacting with values stored in [context.Context].
package ctxutil

import (
	"context"
	"log/slog"

	"github.com/taibuivan/yomira/internal/platform/ctxkey"
	"github.com/taibuivan/yomira/internal/platform/sec"
)

// # Request Tracing

// WithRequestID returns a new context with the provided request ID attached.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxkey.KeyRequestID, id)
}

// GetRequestID retrieves the request ID from the context.
// Returns an empty string if not found.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxkey.KeyRequestID).(string)
	return id
}

// # Structured Logging

// WithLogger returns a new context with the provided logger attached.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxkey.KeyLogger, logger)
}

// GetLogger retrieves the logger from the context.
// If no logger is found, it returns the global default logger.
func GetLogger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(ctxkey.KeyLogger).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}

// # Identity & Access

// WithAuthUser returns a new context with the provided auth claims attached.
func WithAuthUser(ctx context.Context, user *sec.AuthClaims) context.Context {
	return context.WithValue(ctx, ctxkey.KeyUser, user)
}

// GetAuthUser retrieves the [*sec.AuthClaims] from the [context.Context].
func GetAuthUser(ctx context.Context) *sec.AuthClaims {
	claims, ok := ctx.Value(ctxkey.KeyUser).(*sec.AuthClaims)
	if !ok {
		return nil
	}
	return claims
}
