// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package constants provides centralized, immutable values for the entire platform.

It defines default timeouts, rate limits, and cross-cutting keys that are shared
between different layers of the system.

Categories:

  - Server Timing: Read/Write/Idle timeouts for the HTTP server.
  - Rate Limiting: Burst capacities and IP tracking TTLs.
  - Security: JWT issuers and cookie configuration.

Using this package ensures Magic Strings and Magic Numbers are eliminated
from the business logic.
*/
package constants

import "time"

// # Metadata

const (
	AppName    = "yomira-api"
	AppVersion = "0.1.0-dev"
)

// # Server Timing

const (
	// DefaultReadTimeout is the maximum duration for reading the entire request.
	DefaultReadTimeout = 5 * time.Second

	// DefaultWriteTimeout is the maximum duration before timing out writes of the response.
	DefaultWriteTimeout = 10 * time.Second

	// DefaultIdleTimeout is the maximum amount of time to wait for the next request.
	DefaultIdleTimeout = 120 * time.Second

	// DefaultReadHeaderTimeout is the amount of time allowed to read request headers.
	DefaultReadHeaderTimeout = 2 * time.Second

	// GlobalRequestTimeout is the deadline for the entire request lifecycle.
	GlobalRequestTimeout = 30 * time.Second

	// ShutdownTimeout is how long we wait for in-flight requests to complete during shutdown.
	ShutdownTimeout = 30 * time.Second
)

// # Rate Limiting

const (
	// DefaultRateLimitRPS is the requests per second allowed per IP.
	DefaultRateLimitRPS = 100.0

	// DefaultRateLimitBurst is the maximum burst allowed for the rate limiter.
	DefaultRateLimitBurst = 150

	// RateLimitCleanupInterval is how often old IP entries are removed from memory.
	RateLimitCleanupInterval = 1 * time.Minute

	// RateLimitClientTTL is how long a client must be idle before its entry is deleted.
	RateLimitClientTTL = 3 * time.Minute
)

// # Authentication

const (
	// AuthIssuer is the standard 'iss' claim in JWTs.
	AuthIssuer = "yomira.app"

	// ContextKeyUser is the key used to store user claims in the request context.
	ContextKeyUser = "user_claims"

	// RefreshTokenCookieName is the name of the cookie that stores the refresh token.
	RefreshTokenCookieName = "refresh_token"

	// RefreshTokenCookiePath is the scoped path for the refresh token cookie.
	RefreshTokenCookiePath = "/api/v1/auth"
)

// # JSON Field Identifiers

const (
	FieldData    = "data"
	FieldMeta    = "meta"
	FieldError   = "error"
	FieldCode    = "code"
	FieldDetails = "details"
	FieldItems   = "items"
	FieldTotal   = "total"
	FieldMessage = "message"
	FieldStatus  = "status"
	FieldApp     = "app"
	FieldVersion = "version"
	FieldChecks  = "checks"
)

// # Database Schemas

const (
	SchemaCore  = "core"
	SchemaUsers = "users"
)

// # Redis Prefixes (Cache Taxonomy)

const (
	RedisPrefixResetToken  = "auth:reset_token:"
	RedisPrefixVerifyToken = "auth:verify_token:"
	RedisPrefixSession     = "auth:session:"
)
