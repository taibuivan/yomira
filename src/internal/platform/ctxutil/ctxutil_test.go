// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package ctxutil_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/taibuivan/yomira/internal/platform/ctxutil"
	"github.com/taibuivan/yomira/internal/platform/sec"
)

/*
TestContext_RequestID verifies that Request IDs can be injected and retrieved.
*/
func TestContext_RequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-id"

	// 1. Initially should be empty
	assert.Empty(t, ctxutil.GetRequestID(ctx))

	// 2. Inject and retrieve
	ctx = ctxutil.WithRequestID(ctx, requestID)
	assert.Equal(t, requestID, ctxutil.GetRequestID(ctx))
}

/*
TestContext_Logger verifies that a custom logger can be stored in context.
*/
func TestContext_Logger(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 1. Initially should return the default logger
	assert.Equal(t, slog.Default(), ctxutil.GetLogger(ctx))

	// 2. Inject and retrieve
	ctx = ctxutil.WithLogger(ctx, logger)
	assert.Equal(t, logger, ctxutil.GetLogger(ctx))
}

/*
TestContext_AuthUser verifies that AuthClaims can be stored in context.
*/
func TestContext_AuthUser(t *testing.T) {
	ctx := context.Background()
	claims := &sec.AuthClaims{
		UserID: "user-123",
		Role:   "admin",
	}

	// 1. Initially should be nil
	assert.Nil(t, ctxutil.GetAuthUser(ctx))

	// 2. Inject and retrieve
	ctx = ctxutil.WithAuthUser(ctx, claims)
	retrieved := ctxutil.GetAuthUser(ctx)

	assert.NotNil(t, retrieved)
	assert.Equal(t, "user-123", retrieved.UserID)
	assert.Equal(t, "admin", retrieved.Role)
}
