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

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/ctxkey"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/sec"
)

// # Contracts

// TokenVerifier defines the interface needed to verify tokens in middleware.
type TokenVerifier interface {
	// VerifyToken checks if the token string is valid and returns its claims.
	VerifyToken(tokenStr string) (*sec.AuthClaims, error)
}

// # Middleware (Authentication)

// Authenticate extracts and verifies the JWT from the Authorization header.
func Authenticate(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// 1. Extract the Authorization header
			authHeader := request.Header.Get("Authorization")

			// If header is absent, treat as Anonymous Access (downstream will decide if allowed)
			if authHeader == "" {
				next.ServeHTTP(writer, request)
				return
			}

			// 2. Format Validation: Must be "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respond.Error(writer, request, apperr.Unauthorized("Invalid authorization format"))
				return
			}

			// 3. Token Verification via [TokenVerifier]
			tokenStr := parts[1]
			claims, err := verifier.VerifyToken(tokenStr)

			// If verification fails (expired or forged), abort with 401 Unauthorized
			if err != nil {
				respond.Error(writer, request, apperr.Unauthorized("Invalid or expired token"))
				return
			}

			// 4. Context Injection for downstream handlers/services
			context := context.WithValue(request.Context(), ctxkey.KeyUser, claims)
			next.ServeHTTP(writer, request.WithContext(context))
		})
	}
}

// # Middleware (Authorization)

// RequireAuth blocks requests that are not authenticated.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		// Attempt to retrieve user identity from the request context
		claims := GetUser(request.Context())

		// If claims are missing, the user haven't logged in
		if claims == nil {
			respond.Error(writer, request, apperr.Unauthorized("Authentication required"))
			return
		}

		next.ServeHTTP(writer, request)
	})
}

// RequireRole blocks requests if the authenticated user doesn't have the required role.
func RequireRole(role sec.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// Attempt to retrieve user identity from the request context
			claims := GetUser(request.Context())

			// 1. Authentication Check: User MUST be logged in first
			if claims == nil {
				respond.Error(writer, request, apperr.Unauthorized("Authentication required"))
				return
			}

			// 2. Authorization Check: Check if the user's role meets the requirement
			userRole := sec.UserRole(claims.Role)
			if !userRole.AtLeast(role) {
				respond.Error(writer, request, apperr.Forbidden("Insufficient permissions"))
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// # Helpers

// GetUser retrieves the [*sec.AuthClaims] from the [context.Context].
func GetUser(context context.Context) *sec.AuthClaims {
	claims, ok := context.Value(ctxkey.KeyUser).(*sec.AuthClaims)
	if !ok {
		return nil
	}
	return claims
}
