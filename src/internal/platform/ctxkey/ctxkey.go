// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package ctxkey defines typed context keys used by middleware and handlers.
//
// # Safety
//
// It is used to store and retrieve per-request values (user identity, request ID, logger).
// Using a private, unexported type for keys prevents collisions with third-party
// packages that might also use context for storage.
package ctxkey

// key is an unexported type used for context keys to ensure type safety.
//
// # Collision Prevention
//
// Even if another package uses "request_id" as a string key, it will not
// collide with this key type because Go's [context.Context] uses both the
// value AND the type for lookups.
type key string

const (
	// KeyRequestID is the context key for the X-Request-ID correlation value.
	KeyRequestID key = "request_id"

	// KeyUser is the context key for the authenticated user claim ([domain.AuthClaims]).
	KeyUser key = "user"

	// KeyLogger is the context key for the per-request [*log/slog.Logger].
	KeyLogger key = "logger"
)
