// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package middleware provides the cross-cutting HTTP processing chain.

It acts as a series of decorators around the standard http.Handler, injecting
traceability, safety, and security into every request lifecycle.

Standard Stack:

  - Trace: RequestID generation for log correlation.
  - Log: Structured Activity logging (slog).
  - Guard: Rate limiting and CORS validation.
  - Safe: Panic recovery to prevent server crashes.

This package ensures that domain handlers can focus purely on business logic
without worrying about infrastructure-level concerns.
*/
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"github.com/taibuivan/yomira/internal/platform/constants"
	"github.com/taibuivan/yomira/internal/platform/ctxutil"
)

// # Request Tracing

// RequestID attaches a correlation ID to every request for log tracing.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// 1. Check if the client already provided an ID
			requestID := request.Header.Get(constants.HeaderXRequestID)

			// 2. Generate a new one if missing (using UUID v7 for time-sortable properties)
			if requestID == "" {
				uuidV7, err := uuid.NewV7()
				if err != nil {
					requestID = uuid.New().String()
				} else {
					requestID = uuidV7.String()
				}
			}

			// 3. Inject into context and response headers
			ctx := ctxutil.WithRequestID(request.Context(), requestID)
			writer.Header().Set(constants.HeaderXRequestID, requestID)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// No longer needed as it is in ctxutil.

// # Activity Logging

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(code int) {
	recorder.status = code
	recorder.ResponseWriter.WriteHeader(code)
}

// StructuredLogger logs every request status and performance metrics.
// It also injects a request-specific logger into the context.
func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			startTime := time.Now()
			rid := ctxutil.GetRequestID(request.Context())
			ip := RealIP(request)

			// 1. Create a sub-logger for this specific request
			requestLogger := logger.With(
				slog.String("request_id", rid),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.String("ip", ip),
			)

			// 2. Inject this logger into the context for downstream use
			ctx := ctxutil.WithLogger(request.Context(), requestLogger)
			wrappedWriter := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}

			// 3. Proceed to downstream handlers with the enriched context
			next.ServeHTTP(wrappedWriter, request.WithContext(ctx))

			// 4. Final log entry after the request is finished
			latency := time.Since(startTime).Milliseconds()
			logLevel := slog.LevelInfo

			if wrappedWriter.status >= 500 {
				logLevel = slog.LevelError
			} else if wrappedWriter.status >= 400 {
				logLevel = slog.LevelWarn
			}

			// Enlist final response metrics
			logAtters := []any{
				slog.Int("status", wrappedWriter.status),
				slog.Int64("latency_ms", latency),
				slog.String("user_agent", request.UserAgent()),
			}

			// Add user_id if the request is authenticated
			if claims := ctxutil.GetAuthUser(ctx); claims != nil {
				logAtters = append(logAtters, slog.String("user_id", claims.UserID))
			}

			requestLogger.Log(ctx, logLevel, "http_request_finished", logAtters...)
		})
	}
}

// # Rate Limiting

type rateLimitClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*rateLimitClient)
)

// RateLimit limits requests per IP using the token bucket algorithm.
func RateLimit(context context.Context) func(http.Handler) http.Handler {

	// Start a background cleanup routine that respects context cancellation
	go func() {
		ticker := time.NewTicker(constants.RateLimitCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for ip, clientInfo := range clients {
					if time.Since(clientInfo.lastSeen) > constants.RateLimitClientTTL {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			case <-context.Done():
				// Stop the goroutine when the application shuts down
				return
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// Identify the client by their IP address
			clientIP := RealIP(request)

			mu.Lock()
			clientInfo, found := clients[clientIP]

			// Initialize a new limiter if this is a fresh IP
			if !found {
				clientInfo = &rateLimitClient{
					limiter: rate.NewLimiter(
						rate.Limit(constants.DefaultRateLimitRPS),
						constants.DefaultRateLimitBurst,
					),
				}
				clients[clientIP] = clientInfo
			}

			// Update the activity timestamp
			clientInfo.lastSeen = time.Now()

			// Check if the request is allowed by the bucket
			if !clientInfo.limiter.Allow() {
				mu.Unlock()
				writeError(writer, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "Rate limit exceeded")
				return
			}
			mu.Unlock()

			next.ServeHTTP(writer, request)
		})
	}
}

// # Reliability & Safety

// PanicRecovery recovers from panics, logs stack trace, and returns 500.
func PanicRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// Defer a recovery function to catch any runtime exceptions
			defer func() {
				if err := recover(); err != nil {

					// Capture the runtime stack trace for diagnostics
					stackTrace := make([]byte, 2048)
					length := runtime.Stack(stackTrace, false)

					// Retrieve the request-specific logger from context if available
					reqLogger := ctxutil.GetLogger(request.Context())

					// Log the incident to our structured logging system
					reqLogger.ErrorContext(request.Context(), "panic_recovered",
						slog.Any("error", err),
						slog.String("stack", string(stackTrace[:length])),
					)

					// Return a safe, generic error to the client
					writeError(writer, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
				}
			}()

			next.ServeHTTP(writer, request)
		})
	}
}

// # Cross-Origin Resource Sharing

// AppConfig defines the behavior needed by the CORS middleware.
type AppConfig interface {
	IsDevelopment() bool
}

// CORS handles Cross-Origin Resource Sharing based on application environment.
func CORS(cfg AppConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			// 1. Check the Origin header
			origin := request.Header.Get(constants.HeaderOrigin)
			if origin == "" {
				next.ServeHTTP(writer, request)
				return
			}

			// 2. Check if the origin is allowed (strict in PROD, open in DEV)
			isAllowed := false
			if cfg.IsDevelopment() {
				isAllowed = true
			} else {
				if strings.HasSuffix(origin, "yomira.app") {
					isAllowed = true
				}
			}

			// 3. Inject standard CORS headers if authorized
			if isAllowed {
				header := writer.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				header.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization, X-Request-ID")
				header.Set("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Set("Access-Control-Max-Age", "300")
			}

			// 4. Handle pre-flight requests (OPTIONS)
			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// # Middleware Helpers

// RealIP extracts client IP, respecting common proxy headers.
func RealIP(request *http.Request) string {

	// Check standard proxy headers first
	if ip := request.Header.Get(constants.HeaderXRealIP); ip != "" {
		return ip
	}

	if forwarded := request.Header.Get(constants.HeaderXForwardedFor); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	// Fallback to the direct connection's address
	host, _, _ := net.SplitHostPort(request.RemoteAddr)
	return host
}

// writeError outputs a simple JSON error payload.
func writeError(writer http.ResponseWriter, status int, code, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		constants.FieldCode:  code,
		constants.FieldError: message,
	})
}
