// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package middleware provides the HTTP middleware chain for the Yomira API server.
//
// # Architecture
//
// Middleware in this package ensures that all cross-cutting concerns (MDC logging,
// Rate Limiting, CORS, Panic Recovery) are handled consistently before the request
// reaches any business logic or domain handlers.
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

	"github.com/buivan/yomira/internal/platform/constants"
	"github.com/buivan/yomira/internal/platform/ctxkey"
)

// ── 1. RequestID ─────────────────────────────────────────────────────────────

// RequestID attaches a correlation ID to every request for log tracing.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestID := request.Header.Get("X-Request-ID")
			if requestID == "" {
				uuidV7, err := uuid.NewV7()
				if err != nil {
					requestID = uuid.New().String()
				} else {
					requestID = uuidV7.String()
				}
			}

			ctx := context.WithValue(request.Context(), ctxkey.KeyRequestID, requestID)
			writer.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxkey.KeyRequestID).(string)
	return id
}

// ── 2. StructuredLogger ───────────────────────────────────────────────────────

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(code int) {
	recorder.status = code
	recorder.ResponseWriter.WriteHeader(code)
}

// StructuredLogger logs every request status and performance metrics.
func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			startTime := time.Now()
			wrappedWriter := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}

			next.ServeHTTP(wrappedWriter, request)

			latency := time.Since(startTime).Milliseconds()
			logLevel := slog.LevelInfo

			switch {
			case wrappedWriter.status >= 500:
				logLevel = slog.LevelError
			case wrappedWriter.status >= 400:
				logLevel = slog.LevelWarn
			}

			logger.Log(request.Context(), logLevel, "http_request",
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", wrappedWriter.status),
				slog.Int64("latency_ms", latency),
				slog.String("request_id", GetRequestID(request.Context())),
				slog.String("ip", RealIP(request)),
				slog.String("user_agent", request.UserAgent()),
			)
		})
	}
}

// ── 3. RateLimit ─────────────────────────────────────────────────────────────

type rateLimitClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*rateLimitClient)
)

// RateLimit limits requests per IP using the token bucket algorithm.
//
// # Why Token Bucket?
//
// It allows bursts of traffic (e.g., initial page load with many assets) while
// strictly enforcing a long-term rate, which is superior to fixed-window limits.
//
// # Implementation Details
//
// - State is kept in-memory (map).
// - A background goroutine cleans up idle IP trackers to prevent OOM.
func RateLimit() func(http.Handler) http.Handler {
	// Periodic cleanup of idle clients
	go func() {
		ticker := time.NewTicker(constants.RateLimitCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, clientInfo := range clients {
				if time.Since(clientInfo.lastSeen) > constants.RateLimitClientTTL {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			clientIP := RealIP(request)

			mu.Lock()
			clientInfo, found := clients[clientIP]
			if !found {
				clientInfo = &rateLimitClient{
					limiter: rate.NewLimiter(
						rate.Limit(constants.DefaultRateLimitRPS),
						constants.DefaultRateLimitBurst,
					),
				}
				clients[clientIP] = clientInfo
			}
			clientInfo.lastSeen = time.Now()

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

// ── 4. PanicRecovery ─────────────────────────────────────────────────────────

// PanicRecovery recovers from panics, logs stack trace, and returns 500.
//
// # Safety
//
// In Go, an unhandled panic in an HTTP handler will kill the connection and
// print to standard error, but the server stays alive. We catch it here to:
//  1. Prevent leaking stack traces to the client.
//  2. Output the stack trace into our structured logging system for alerts.
func PanicRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stackTrace := make([]byte, 2048)
					length := runtime.Stack(stackTrace, false)

					logger.ErrorContext(request.Context(), "panic_recovered",
						slog.Any("error", err),
						slog.String("stack", string(stackTrace[:length])),
						slog.String("request_id", GetRequestID(request.Context())),
					)

					writeError(writer, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}

// ── 5. CORS ───────────────────────────────────────────────────────────────────

// AppConfig defines the behavior needed by the CORS middleware.
type AppConfig interface {
	IsDevelopment() bool
}

// CORS handles Cross-Origin Resource Sharing based on application environment.
func CORS(cfg AppConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(writer, request)
				return
			}

			isAllowed := false
			if cfg.IsDevelopment() {
				isAllowed = true
			} else {
				if strings.HasSuffix(origin, "yomira.app") {
					isAllowed = true
				}
			}

			if isAllowed {
				header := writer.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				header.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization, X-Request-ID")
				header.Set("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Set("Access-Control-Max-Age", "300")
			}

			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// RealIP extracts client IP, respecting common proxy headers.
func RealIP(request *http.Request) string {
	if ip := request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, _ := net.SplitHostPort(request.RemoteAddr)
	return host
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"code":  code,
		"error": message,
	})
}
