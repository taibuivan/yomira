// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package middleware provides the HTTP middleware chain for the Yomira API server.
//
// # Architecture
//
// Middleware intercepts incoming HTTP requests to apply global policies
// before they reach the domain handlers. This includes cross-cutting concerns
// like Logging, AuthZ/AuthN, Rate Limiting, and CORS.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/buivan/yomira/internal/auth"
	"github.com/buivan/yomira/internal/platform/apperr"
	"github.com/buivan/yomira/internal/platform/ctxkey"
	"github.com/buivan/yomira/internal/platform/respond"
	"github.com/buivan/yomira/internal/platform/sec"
)

// TokenVerifier defines the interface needed to verify tokens in middleware.
//
// # Why an interface?
//
// Defining TokenVerifier here decouples the middleware from the `auth` service
// implementation, allowing us to easily inject mocks during unit testing.
type TokenVerifier interface {
	VerifyToken(tokenStr string) (*sec.AuthClaims, error)
}

// Authenticate extracts and verifies the JWT from the Authorization header.
//
// # Flow
//  1. Check for 'Authorization: Bearer <token>' header.
//  2. If absent, request proceeds as anonymous.
//  3. If present, parse and verify the JWT via [TokenVerifier].
//  4. Inject [*sec.AuthClaims] into the request context for downstream use.
//
// # Parameters
//   - verifier: The TokenVerifier instance.
//
// # Returns
//   - An [http.Handler] middleware.
func Authenticate(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authHeader := request.Header.Get("Authorization")

			// ── 1. Anonymous Access ───────────────────────────────────────────
			if authHeader == "" {
				next.ServeHTTP(writer, request)
				return
			}

			// ── 2. Format Validation ──────────────────────────────────────────
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respond.Error(writer, request, apperr.Unauthorized("Invalid authorization format"))
				return
			}

			// ── 3. Token Verification ─────────────────────────────────────────
			tokenStr := parts[1]
			claims, err := verifier.VerifyToken(tokenStr)
			if err != nil {
				respond.Error(writer, request, apperr.Unauthorized("Invalid or expired token"))
				return
			}

			// ── 4. Context Injection ──────────────────────────────────────────
			ctx := context.WithValue(request.Context(), ctxkey.KeyUser, claims)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// RequireAuth blocks requests that are not authenticated.
//
// # Usage
//
// Must be registered in the router AFTER [Authenticate].
//
// # Flow
//  1. Check if [*sec.AuthClaims] exists in context.
//  2. If missing, abort with HTTP 401 Unauthorized.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		claims := GetUser(request.Context())
		if claims == nil {
			respond.Error(writer, request, apperr.Unauthorized("Authentication required"))
			return
		}
		next.ServeHTTP(writer, request)
	})
}

// RequireRole blocks requests if the authenticated user doesn't have the required role.
//
// # Usage
//
// Must be registered in the router AFTER [Authenticate]. It automatically implies
// [RequireAuth] so you don't need to mount both.
//
// # Flow
//  1. Check if [*sec.AuthClaims] exists in context (implies AuthN).
//  2. Check if the user's role meets or exceeds the required target role using [auth.UserRole.AtLeast].
//  3. If insufficient, abort with HTTP 403 Forbidden.
func RequireRole(role auth.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			claims := GetUser(request.Context())

			// ── 1. Authentication Check ───────────────────────────────────────
			if claims == nil {
				respond.Error(writer, request, apperr.Unauthorized("Authentication required"))
				return
			}

			// ── 2. Authorization Check ────────────────────────────────────────
			userRole := auth.UserRole(claims.Role)
			if !userRole.AtLeast(role) {
				respond.Error(writer, request, apperr.Forbidden("Insufficient permissions"))
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// GetUser retrieves the [*sec.AuthClaims] from the [context.Context].
//
// # Returns
//   - A pointer to [*sec.AuthClaims] if the user is authenticated.
//   - nil if the user is anonymous.
func GetUser(ctx context.Context) *sec.AuthClaims {
	claims, ok := ctx.Value(ctxkey.KeyUser).(*sec.AuthClaims)
	if !ok {
		return nil
	}
	return claims
}
