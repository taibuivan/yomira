# Go Middleware Reference — Yomira Backend

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Applies to:** `src/server/middleware/`

---

## Table of Contents

1. [Middleware Stack Overview](#1-middleware-stack-overview)
2. [Request ID](#2-request-id)
3. [Structured Logging](#3-structured-logging)
4. [Panic Recovery](#4-panic-recovery)
5. [CORS](#5-cors)
6. [Maintenance Mode](#6-maintenance-mode)
7. [Rate Limiter](#7-rate-limiter)
8. [Authentication (JWT)](#8-authentication-jwt)
9. [Authorization (Role Guard)](#9-authorization-role-guard)
10. [Content-Type Enforcer](#10-content-type-enforcer)
11. [Implementation Notes](#11-implementation-notes)

---

## 1. Middleware Stack Overview

Middleware is applied in a strict order — **outer wraps inner**. Each request passes through the chain top-to-bottom; responses flow back bottom-to-top.

```
Incoming Request
       │
       ▼
┌─────────────────────────┐
│  1. RequestID           │  Attach / generate X-Request-ID
├─────────────────────────┤
│  2. StructuredLogger    │  Log method, path, status, latency
├─────────────────────────┤
│  3. PanicRecovery       │  Catch panics → 500 response
├─────────────────────────┤
│  4. CORS                │  Preflight + headers
├─────────────────────────┤
│  5. MaintenanceMode     │  503 if site.maintenance_mode = true
├─────────────────────────┤
│  6. RateLimiter         │  Redis sliding window per IP / user
├─────────────────────────┤
│  7. Auth (optional)     │  Parse Bearer JWT, set ctx.User
├─────────────────────────┤
│  8. RoleGuard (optional)│  Assert minimum role on protected routes
├─────────────────────────┤
│  9. ContentType         │  Reject non-JSON POST/PATCH/PUT bodies
├─────────────────────────┤
│       Handler           │
└─────────────────────────┘
       │
       ▼
Outgoing Response
```

### Global vs route-level

| Middleware | Applied | Notes |
|---|---|---|
| RequestID | Global — all routes | Always first |
| StructuredLogger | Global — all routes | After RequestID so ID is logged |
| PanicRecovery | Global — all routes | Must wrap all others |
| CORS | Global — all routes | Must be before Auth |
| MaintenanceMode | Global — all routes | Skips admin routes |
| RateLimiter | Global (per-scope) | Different limits per route group |
| Auth | Route group: authenticated routes | Optional — some routes public |
| RoleGuard | Route level: admin/mod endpoints | Requires Auth |
| ContentType | Route group: write routes | POST, PATCH, PUT, DELETE with body |

---

## 2. Request ID

**Purpose:** Attach a correlation ID to every request for distributed tracing and log linking.

```go
// middleware/requestid.go
func RequestID() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            id := r.Header.Get("X-Request-ID")
            if id == "" {
                id = uuidv7.New().String()   // generate if client didn't send one
            }
            ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
            w.Header().Set("X-Request-ID", id)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Behavior:**
- If client sends `X-Request-ID`: use it as-is (trust client for tracing)
- If absent: generate a new UUIDv7
- Always echoed in response header `X-Request-ID`

**Context key:** `ctxKeyRequestID` — retrieve with `GetRequestID(ctx)`

---

## 3. Structured Logging

**Purpose:** Log every request with method, path, status code, latency, and request ID. Uses `log/slog` (Go 1.21+).

```go
// middleware/logger.go
func StructuredLogger(log *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := NewStatusRecorder(w)      // wraps ResponseWriter to capture status

            next.ServeHTTP(ww, r)

            log.InfoContext(r.Context(), "request",
                slog.String("method",     r.Method),
                slog.String("path",       r.URL.Path),
                slog.Int("status",        ww.Status()),
                slog.Int64("latency_ms",  time.Since(start).Milliseconds()),
                slog.String("request_id", GetRequestID(r.Context())),
                slog.String("ip",         RealIP(r)),
                slog.String("user_agent", r.UserAgent()),
            )
        })
    }
}
```

**Log levels:**
| Status range | Log level |
|---|---|
| `2xx` | `INFO` |
| `3xx` | `INFO` |
| `4xx` | `WARN` |
| `5xx` | `ERROR` |

**Fields always logged:**
- `method`, `path`, `status`, `latency_ms`, `request_id`, `ip`, `user_agent`
- `user_id` — added after Auth middleware parses the JWT (via context)

---

## 4. Panic Recovery

**Purpose:** Catch any unhandled `panic()` in handlers, log the stack trace, and return `500 Internal Server Error` instead of crashing the process.

```go
// middleware/recovery.go
func PanicRecovery(log *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if rec := recover(); rec != nil {
                    buf := make([]byte, 4096)
                    n := runtime.Stack(buf, false)

                    log.ErrorContext(r.Context(), "panic recovered",
                        slog.Any("panic", rec),
                        slog.String("stack", string(buf[:n])),
                        slog.String("request_id", GetRequestID(r.Context())),
                    )

                    // Do not expose internals to client
                    WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
                        "An unexpected error occurred")
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

**Important:** PanicRecovery must wrap **all other middleware** — it is the outermost layer after RequestID and Logger.

---

## 5. CORS

**Purpose:** Set Cross-Origin Resource Sharing headers and handle browser preflight `OPTIONS` requests.

```go
// middleware/cors.go
var allowedOrigins = map[string]bool{
    "https://yomira.app":         true,
    "https://www.yomira.app":     true,
    "https://admin.yomira.app":   true,
    // Dev origins loaded from config:
    "http://localhost:3000":       true,
    "http://localhost:5173":       true,
}

func CORS(cfg *config.Config) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if allowedOrigins[origin] {
                w.Header().Set("Access-Control-Allow-Origin",  origin)
                w.Header().Set("Vary", "Origin")   // critical for caching
            }
            w.Header().Set("Access-Control-Allow-Methods",
                "GET, POST, PUT, PATCH, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers",
                "Authorization, Content-Type, X-Request-ID")
            w.Header().Set("Access-Control-Expose-Headers",
                "X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, X-Request-ID")
            w.Header().Set("Access-Control-Max-Age", "86400")

            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)  // 204 for preflight
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Note:** `Vary: Origin` must be set whenever `Access-Control-Allow-Origin` is dynamic — otherwise CDN/proxies may cache incorrect origin headers.

---

## 6. Maintenance Mode

**Purpose:** When `system.setting["site.maintenance_mode"] = "true"`, return `503 Service Unavailable` for all non-admin requests.

```go
// middleware/maintenance.go
func MaintenanceMode(settings *config.Settings) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if settings.GetBool("site.maintenance_mode") {
                // Admin users bypass maintenance mode
                user := GetUserFromContext(r.Context())
                if user == nil || user.Role != "admin" {
                    w.Header().Set("Retry-After", "3600")
                    WriteError(w, http.StatusServiceUnavailable,
                        "SERVICE_UNAVAILABLE",
                        "Yomira is currently under maintenance. Please try again later.")
                    return
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Note:** Settings are loaded into memory at startup (see `SYSTEM_API.md`). `PUT /admin/settings/site.maintenance_mode` triggers an in-memory reload signal — no restart required.

---

## 7. Rate Limiter

**Purpose:** Enforce request rate limits using Redis sliding window counters. Applies different limits to different route groups.

```go
// middleware/ratelimit.go
type RateLimit struct {
    Requests int           // max requests
    Window   time.Duration // time window
    KeyFunc  func(r *http.Request) string  // how to identify the caller
}

func RateLimiter(redis *redis.Client, limit RateLimit) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := "ratelimit:" + limit.KeyFunc(r)
            now := time.Now().UnixMilli()
            windowStart := now - limit.Window.Milliseconds()

            // Sliding window via Redis sorted set
            pipe := redis.Pipeline()
            pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
            pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
            pipe.ZCard(ctx, key)
            pipe.Expire(ctx, key, limit.Window)
            results, _ := pipe.Exec(ctx)

            count := results[2].(*redis.IntCmd).Val()
            remaining := int64(limit.Requests) - count
            reset := time.Now().Add(limit.Window).Unix()

            w.Header().Set("X-RateLimit-Limit",     strconv.Itoa(limit.Requests))
            w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(max(remaining, 0), 10))
            w.Header().Set("X-RateLimit-Reset",     strconv.FormatInt(reset, 10))

            if count > int64(limit.Requests) {
                w.Header().Set("Retry-After", strconv.FormatInt(int64(limit.Window.Seconds()), 10))
                WriteError(w, http.StatusTooManyRequests, "RATE_LIMITED",
                    fmt.Sprintf("Too many requests. Please try again in %ds.", int(limit.Window.Seconds())))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Rate limit profiles (keyed by route group)

```go
// router.go — applying different limits to different route groups
r.Group(func(r chi.Router) {
    // Auth endpoints — strict
    r.Use(RateLimiter(redis, RateLimit{
        Requests: 10, Window: 15 * time.Minute,
        KeyFunc: ByIP,
    }))
    r.Post("/auth/register", authHandler.Register)
    r.Post("/auth/login",    authHandler.Login)
})

r.Group(func(r chi.Router) {
    // General authenticated endpoints
    r.Use(Auth(jwt))
    r.Use(RateLimiter(redis, RateLimit{
        Requests: 300, Window: time.Minute,
        KeyFunc: ByUserID,   // if unauthenticated, fall back to IP
    }))
    // ... routes
})
```

### Key functions

```go
func ByIP(r *http.Request) string {
    return "ip:" + RealIP(r)
}

func ByUserID(r *http.Request) string {
    if user := GetUserFromContext(r.Context()); user != nil {
        return "user:" + user.ID
    }
    return "ip:" + RealIP(r)  // fallback for unauthenticated
}

func ByEmail(email string) string {
    return "email:" + strings.ToLower(email)
}
```

### `RealIP` — handle proxies

```go
func RealIP(r *http.Request) string {
    // Trust X-Forwarded-For from trusted proxy only
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
        return strings.Split(fwd, ",")[0]    // leftmost = client IP
    }
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}
```

---

## 8. Authentication (JWT)

**Purpose:** Parse the `Authorization: Bearer <token>` header, validate the JWT RS256 signature and expiry, and attach the user claim to the request context.

```go
// middleware/auth.go
func Auth(verifier *jwt.Verifier) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            header := r.Header.Get("Authorization")
            if header == "" || !strings.HasPrefix(header, "Bearer ") {
                WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED",
                    "Missing or invalid Authorization header")
                return
            }

            raw := strings.TrimPrefix(header, "Bearer ")
            claims, err := verifier.Verify(raw)  // RS256 signature + exp check
            if err != nil {
                if errors.Is(err, jwt.ErrExpired) {
                    WriteError(w, http.StatusUnauthorized, "TOKEN_EXPIRED",
                        "Access token has expired. Please refresh.")
                    return
                }
                WriteError(w, http.StatusUnauthorized, "TOKEN_INVALID",
                    "Invalid access token")
                return
            }

            ctx := context.WithValue(r.Context(), ctxKeyUser, &AuthUser{
                ID:         claims.Subject,
                Role:       claims.Role,
                IsVerified: claims.IsVerified,
            })
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// OptionalAuth — sets ctx.User if token present, but doesn't reject missing tokens.
// Use on public endpoints that return personalised data when authenticated.
func OptionalAuth(verifier *jwt.Verifier) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            header := r.Header.Get("Authorization")
            if header != "" && strings.HasPrefix(header, "Bearer ") {
                raw := strings.TrimPrefix(header, "Bearer ")
                if claims, err := verifier.Verify(raw); err == nil {
                    ctx := context.WithValue(r.Context(), ctxKeyUser, &AuthUser{
                        ID: claims.Subject, Role: claims.Role,
                    })
                    r = r.WithContext(ctx)
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Context helpers

```go
func GetUserFromContext(ctx context.Context) *AuthUser {
    u, _ := ctx.Value(ctxKeyUser).(*AuthUser)
    return u   // nil if unauthenticated
}

func MustGetUser(ctx context.Context) *AuthUser {
    u := GetUserFromContext(ctx)
    if u == nil {
        panic("MustGetUser called on unauthenticated request")
    }
    return u
}
```

### JWT Public Key Loading

```go
// At startup — load RS256 public key for verification
func NewVerifier(cfg *config.Config) (*jwt.Verifier, error) {
    raw, err := os.ReadFile(cfg.JWTPublicKeyPath)
    if err != nil {
        // Fallback: parse from env var
        raw = []byte(cfg.JWTPublicKey)
    }
    block, _ := pem.Decode(raw)
    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    return jwt.NewVerifier(pub.(*rsa.PublicKey))
}
```

---

## 9. Authorization (Role Guard)

**Purpose:** Assert the authenticated user has at least the required role. Applied per-route after `Auth`.

```go
// middleware/authz.go

// Role hierarchy: admin(3) > moderator(2) > member(1) > banned(0)
var roleRank = map[string]int{
    "admin":     3,
    "moderator": 2,
    "member":    1,
    "banned":    0,
}

func RequireRole(minRole string) func(http.Handler) http.Handler {
    required := roleRank[minRole]
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUserFromContext(r.Context())
            if user == nil {
                WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED",
                    "Authentication required")
                return
            }
            if roleRank[user.Role] < required {
                WriteError(w, http.StatusForbidden, "FORBIDDEN",
                    "Insufficient permissions")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Convenience wrappers
var (
    RequireAdmin = RequireRole("admin")
    RequireMod   = RequireRole("moderator")
    RequireMember = RequireRole("member")
)

// RequireVerified — member must have confirmed email
func RequireVerified(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := MustGetUser(r.Context())
        if !user.IsVerified {
            WriteError(w, http.StatusForbidden, "EMAIL_NOT_VERIFIED",
                "Please verify your email address before performing this action")
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Usage in router

```go
// Public — no auth
r.Get("/comics/{id}", comicHandler.GetByID)

// Authenticated member required
r.With(Auth(jwt), RequireMember).
    Post("/comics/{id}/comments", commentHandler.Create)

// Verified member required
r.With(Auth(jwt), RequireMember, RequireVerified).
    Post("/comics/{id}/rating", ratingHandler.Upsert)

// Admin only
r.With(Auth(jwt), RequireAdmin).
    Delete("/admin/users/{id}", adminHandler.DeleteUser)

// Admin or moderator
r.With(Auth(jwt), RequireMod).
    Patch("/admin/reports/{id}", reportHandler.Resolve)
```

---

## 10. Content-Type Enforcer

**Purpose:** Reject requests with bodies that are not `application/json`. Prevents form-submission attacks and client misconfiguration.

```go
// middleware/contenttype.go
func RequireJSON(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only check methods that typically have bodies
        if r.Method == http.MethodPost ||
            r.Method == http.MethodPatch ||
            r.Method == http.MethodPut {
            ct := r.Header.Get("Content-Type")
            if !strings.HasPrefix(ct, "application/json") {
                WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR",
                    "Content-Type must be application/json")
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

**Exceptions:** Upload endpoints (`PUT /me/avatar`, `POST /admin/chapters/:id/pages`) use `multipart/form-data` — they skip this middleware.

---

## 11. Implementation Notes

### WriteError helper

```go
// Used by all middleware to write consistent error responses
func WriteError(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(map[string]string{
        "error": message,
        "code":  code,
    })
}
```

### Middleware context keys

```go
// shared/ctxkey/keys.go — prevent collisions with typed keys
type ctxKey string

const (
    ctxKeyRequestID ctxKey = "request_id"
    ctxKeyUser      ctxKey = "user"
)
```

### Recommended router: chi

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()

// Global middleware — order matters
r.Use(middleware.RequestID())
r.Use(middleware.StructuredLogger(log))
r.Use(middleware.PanicRecovery(log))
r.Use(middleware.CORS(cfg))
r.Use(middleware.MaintenanceMode(settings))

// Route-group-specific middleware applied inline (see §7, §8, §9)
```

### Middleware testing

```go
func TestAuthMiddleware_MissingToken(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/me", nil)
    rr := httptest.NewRecorder()

    handler := Auth(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    handler.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusUnauthorized, rr.Code)
    var body map[string]string
    json.NewDecoder(rr.Body).Decode(&body)
    assert.Equal(t, "UNAUTHORIZED", body["code"])
}
```
