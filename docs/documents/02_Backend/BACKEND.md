# Yomira — Backend Development Guide

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.1.0 — 2026-02-21  
> **Status:** Active

---

## Table of Contents

1. [Go Package Philosophy](#1-go-package-philosophy)
2. [Project Structure](#2-project-structure)
3. [Phase 1.1 — Project Setup](#31-project-setup)
4. [Phase 1.2 — Configuration](#32-configuration)
5. [Phase 1.3 — Domain Layer](#33-domain-layer)
6. [Phase 1.4 — Storage Layer](#34-storage-layer)
7. [Phase 1.5 — Authentication](#35-authentication)
8. [Phase 1.6 — API & Routing](#36-api--routing)
9. [Phase 1.7 — HTML Templates](#37-html-templates)
10. [API Contract Reference](#4-api-contract-reference)
11. [Error Handling](#5-error-handling)
12. [Background Workers](#6-background-workers)

---

## 1. Go Package Philosophy

### Why there are no `utils/`, `enums/`, or `types/` folders

This is the most common misconception when developers come from Java or other OOP-heavy languages.

**Java organizes by what kind of thing it is:**
```
enums/Status.java        → all enumerations
utils/StringUtil.java    → utility functions
types/PaginationReq.java → shared type definitions
interfaces/IRepo.java    → interface declarations
```

**Go organizes by what it belongs to — by domain and responsibility:**
```
Java →  Go
enums/Status.java        →  internal/domain/comic.go     (Status const lives here)
utils/StringUtil.java    →  internal/comic/format.go     (used only in comic package)
types/PaginationReq      →  internal/api/request.go      (defined where it's consumed)
interfaces/IRepo.java    →  internal/domain/repo.go      (interface lives with its consumer)
```

**The core rule:** A package is a unit of responsibility, not a bucket for a category of things.

| Question | Answer |
|---|---|
| Where do enums go? | Go has no enums. Use `const` blocks (with `iota` or typed string constants) inside the package that owns the concept. `comic.Status` lives in the `comic` package. |
| Where do utils go? | A `utils` package is a code smell. Create a specific package (`pagination`, `slug`) or keep the function in the package that uses it. |
| Where do types go? | Types belong to the domain they define. `Comic`, `Chapter`, `User` → `internal/domain`. Request/response shapes → `internal/api`. |
| Where do interfaces go? | In Go, interfaces are defined **where they are consumed**, not where they are implemented. A `ComicRepository` interface belongs in `internal/domain`, not in an `interfaces/` folder. |
| Where do constants go? | With the package they relate to. App-wide config values → `internal/config`. |

---

## 2. Project Structure

```
yomira/
├── cmd/
│   └── api/
│       └── main.go             # Entry point: wire dependencies, start server
│
├── internal/
│   ├── config/
│   │   └── config.go           # Config struct, reads env vars once at startup
│   │
│   ├── domain/                 # Pure Go — no external dependencies
│   │   ├── comic.go            # Comic, Chapter, Page structs + repository interfaces
│   │   ├── user.go             # User, Session structs + repository interfaces
│   │   ├── library.go          # Entry, Progress, CustomList structs
│   │   ├── social.go           # Comment, Rating, Notification structs
│   │   └── crawler.go          # Source, Job structs
│   │
│   ├── service/                # Business logic — depends on domain interfaces only
│   │   ├── comic.go            # ComicService: list, search, filter, detail
│   │   ├── user.go             # UserService: register, profile, follow
│   │   ├── library.go          # LibraryService: shelf, progress, lists
│   │   ├── social.go           # SocialService: comments, ratings, feed
│   │   └── crawler.go          # CrawlerService: schedule, run, parse
│   │
│   ├── storage/                # Implements domain interfaces
│   │   ├── postgres/
│   │   │   ├── db.go           # pgxpool setup, migrations
│   │   │   ├── comic_repo.go
│   │   │   ├── user_repo.go
│   │   │   ├── library_repo.go
│   │   │   ├── social_repo.go
│   │   │   └── crawler_repo.go
│   │   └── redis/
│   │       ├── session_store.go  # Refresh token store
│   │       └── counter.go        # viewcount INCR + flush
│   │
│   ├── auth/
│   │   ├── service.go          # Login, register, token issuance
│   │   ├── middleware.go       # HTTP middleware: validate JWT, inject user ctx
│   │   └── token.go            # JWT sign/verify (RS256)
│   │
│   ├── api/
│   │   ├── server.go           # Router setup, middleware chain
│   │   ├── response.go         # Shared JSON response helpers
│   │   ├── request.go          # Shared request parsing helpers
│   │   ├── handler_auth.go     # POST /api/v1/auth/login, logout, refresh
│   │   ├── handler_comic.go    # GET  /api/v1/comics, /comics/:id
│   │   ├── handler_chapter.go  # GET  /api/v1/chapters/:id
│   │   ├── handler_library.go  # GET/PUT /api/v1/me/library
│   │   ├── handler_social.go   # Comments, ratings, feed
│   │   └── handler_admin.go    # Admin-only endpoints
│   │
│   ├── worker/                 # Background goroutines
│   │   ├── counter_flush.go    # Flush Redis viewcount → PostgreSQL every 60s
│   │   ├── hasnew.go           # Update library.entry.hasnew after crawler
│   │   └── partition.go        # Create next month's partitions (monthly cron)
│   │
│   └── crawler/
│       ├── scheduler.go        # Job queue + rate limiter
│       ├── extension.go        # Plugin loader
│       └── extensions/         # Per-source plugins
│           ├── nettruyen/
│           └── mangadex/
│
├── web/
│   ├── templates/              # Go html/template files
│   │   ├── base.html           # Base layout: nav, head, footer
│   │   ├── home.html
│   │   ├── comic.html          # Comic detail page
│   │   ├── reader.html         # Chapter reader
│   │   ├── library.html
│   │   ├── login.html
│   │   └── profile.html
│   └── static/
│       ├── css/
│       ├── js/
│       └── img/
│
├── migrations/
│   ├── 000001_setup.up.sql
│   ├── 000001_setup.down.sql
│   └── ...
│
├── src/common/DML/             # Schema source of truth
│   ├── 00_SETUP/
│   ├── 10_USERS/
│   └── ...
│
├── .env
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

---

## 3.1 Project Setup

**`go.mod`** — initialize with a meaningful module path:
```
module github.com/buivan/yomira
go 1.22
```

**`cmd/api/main.go`** — entry point does exactly three things:
1. Load configuration from env.
2. Initialize dependencies (database pool, Redis, services).
3. Start the HTTP server.

No business logic lives here. `main.go` is a wiring file only.

**`.env`** — environment-specific values, never committed:
```env
DATABASE_URL=postgres://yomira:password@localhost:5432/yomira?sslmode=disable
REDIS_URL=redis://localhost:6379
SESSION_SECRET=your-32-byte-secret-here
SERVER_PORT=8080
ENVIRONMENT=development
DEBUG=true
JWT_PRIVATE_KEY_PATH=./keys/private.pem
JWT_PUBLIC_KEY_PATH=./keys/public.pem
S3_BUCKET=yomira-media
S3_REGION=auto
S3_ENDPOINT=https://your-account.r2.cloudflarestorage.com
```

**`Makefile`** — common dev commands:
```makefile
run:
    go run ./cmd/api

migrate-up:
    migrate -database ${DATABASE_URL} -path ./migrations up

migrate-down:
    migrate -database ${DATABASE_URL} -path ./migrations down 1

test:
    go test ./... -race -count=1

lint:
    golangci-lint run
```

---

## 3.2 Configuration

A single `Config` struct populated once at startup:

```go
// internal/config/config.go
// @author tai.buivan.jp@gmail.com

type Config struct {
    DatabaseURL    string        `env:"DATABASE_URL,required"`
    RedisURL       string        `env:"REDIS_URL,required"`
    ServerPort     string        `env:"SERVER_PORT" envDefault:"8080"`
    Environment    string        `env:"ENVIRONMENT" envDefault:"development"`
    Debug          bool          `env:"DEBUG" envDefault:"false"`
    SessionSecret  string        `env:"SESSION_SECRET,required"`
    JWTPrivKeyPath string        `env:"JWT_PRIVATE_KEY_PATH,required"`
    JWTPubKeyPath  string        `env:"JWT_PUBLIC_KEY_PATH,required"`
    S3Bucket       string        `env:"S3_BUCKET"`
    S3Endpoint     string        `env:"S3_ENDPOINT"`
}

func Load() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("config: %w", err)
    }
    return cfg, nil
}
```

**Why a dedicated package?** Centralizing config prevents scattered `os.Getenv()` calls. Any package that needs config receives a `*Config` via dependency injection — it never reads env vars directly.

---

## 3.3 Domain Layer

The domain layer contains pure Go structs and interfaces. It has **zero external dependencies**.

```go
// internal/domain/comic.go
// @author tai.buivan.jp@gmail.com

type ComicStatus string

const (
    ComicStatusOngoing   ComicStatus = "ongoing"
    ComicStatusCompleted ComicStatus = "completed"
    ComicStatusHiatus    ComicStatus = "hiatus"
    ComicStatusCancelled ComicStatus = "cancelled"
    ComicStatusUnknown   ComicStatus = "unknown"
)

func (s ComicStatus) IsValid() bool {
    switch s {
    case ComicStatusOngoing, ComicStatusCompleted,
         ComicStatusHiatus, ComicStatusCancelled, ComicStatusUnknown:
        return true
    }
    return false
}

type Comic struct {
    ID             string
    Title          string
    TitleAlt       []string
    Slug           string
    Synopsis       string
    CoverURL       string
    Status         ComicStatus
    ContentRating  ContentRating
    Demographic    Demographic
    DefaultReadMode ReadMode
    OriginLanguage string
    Year           *int16
    Links          map[string]string
    ViewCount      int64
    FollowCount    int64
    RatingAvg      float64
    RatingBayesian float64
    RatingCount    int
    IsLocked       bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      *time.Time
}

type ComicRepository interface {
    List(ctx context.Context, f ComicFilter) ([]*Comic, int, error)
    FindByID(ctx context.Context, id string) (*Comic, error)
    FindBySlug(ctx context.Context, slug string) (*Comic, error)
    Search(ctx context.Context, query string, f ComicFilter) ([]*Comic, int, error)
    Create(ctx context.Context, c *Comic) error
    Update(ctx context.Context, c *Comic) error
    SoftDelete(ctx context.Context, id string) error
}
```

**Key point:** Repository interfaces are defined in `domain`, not in `storage`. The `storage` package implements these interfaces. This keeps `domain` independent of any library.

---

## 3.4 Storage Layer

Implements repository interfaces. All SQL lives here. No ORM.

```go
// internal/storage/postgres/comic_repo.go

type comicRepo struct {
    pool *pgxpool.Pool
}

func (r *comicRepo) FindByID(ctx context.Context, id string) (*domain.Comic, error) {
    const q = `
        SELECT id, title, titlealt, slug, synopsis, coverurl,
               status, contentrating, demographic, defaultreadmode,
               originlanguage, year, links, viewcount, followcount,
               ratingavg, ratingbayesian, ratingcount,
               islocked, createdat, updatedat, deletedat
        FROM core.comic
        WHERE id = $1 AND deletedat IS NULL`

    row := r.pool.QueryRow(ctx, q, id)
    return scanComic(row)
}
```

**Conventions:**
- All queries are parameterized (`$1`, `$2`...) — never string concatenation.
- Use `pgxpool.Pool` for connection pooling, not a single connection.
- Complex queries (search + filters) use a query builder pattern — not an ORM.
- Transactions for multi-step writes: `pool.BeginTx(ctx, pgx.TxOptions{})`.

---

## 3.5 Authentication

### Token flow

```
Login:
  POST /api/v1/auth/login
  │  → validate credentials
  │  → issue access_token (JWT, 15 min) + refresh_token (opaque, 30 days)
  │  → store SHA-256(refresh_token) in users.session
  └► Set-Cookie: refresh_token=...; HttpOnly; SameSite=Strict

Refresh:
  POST /api/v1/auth/refresh
  │  → read refresh_token cookie
  │  → lookup SHA-256(token) in users.session
  │  → validate not expired, not revoked
  └► Return new access_token (+ rotate refresh_token)

Protected request:
  GET /api/v1/me/library
  Authorization: Bearer {access_token}
  │  → middleware validates JWT signature + expiry
  │  → extract userID + role from claims
  │  → inject into context.Context
  └► handler reads: auth.UserFromCtx(ctx)
```

### Middleware

```go
// internal/auth/middleware.go

func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            api.Error(w, http.StatusUnauthorized, "missing token")
            return
        }
        claims, err := validateJWT(token)
        if err != nil {
            api.Error(w, http.StatusUnauthorized, "invalid token")
            return
        }
        ctx := WithUser(r.Context(), claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 3.6 API & Routing

### Route registration (chi router)

```go
// internal/api/server.go

r := chi.NewRouter()

// Global middleware
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(corsMiddleware)

// Static files
r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

// Public API
r.Group(func(r chi.Router) {
    r.Post("/api/v1/auth/login",   h.auth.Login)
    r.Post("/api/v1/auth/refresh", h.auth.Refresh)
    r.Get("/api/v1/comics",        h.comic.List)
    r.Get("/api/v1/comics/{id}",   h.comic.Detail)
})

// Protected API
r.Group(func(r chi.Router) {
    r.Use(auth.RequireAuth)
    r.Post("/api/v1/auth/logout",          h.auth.Logout)
    r.Get("/api/v1/me/library",            h.library.List)
    r.Put("/api/v1/me/library/{comicId}",  h.library.Upsert)
})

// HTML pages (SSR)
r.Get("/",            h.page.Home)
r.Get("/title/{id}", h.page.ComicDetail)
r.Get("/login",      h.page.Login)
```

### Response helpers

```go
// internal/api/response.go

// All API responses follow this envelope:
// Success:  { "data": <payload>, "meta": { ... } }
// Error:    { "error": "message", "code": "ERROR_CODE" }

func JSON(w http.ResponseWriter, status int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, data any) {
    JSON(w, http.StatusOK, envelope{"data": data})
}

func Paginated(w http.ResponseWriter, data any, total, page, limit int) {
    JSON(w, http.StatusOK, envelope{
        "data": data,
        "meta": map[string]int{"total": total, "page": page, "limit": limit},
    })
}

func Error(w http.ResponseWriter, status int, message string) {
    JSON(w, status, envelope{"error": message})
}
```

---

## 3.7 HTML Templates

Go's `html/template` handles server-side rendering.

**`base.html`** — layout skeleton:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}Yomira{{end}}</title>
    <link rel="stylesheet" href="/static/css/style.css">
</head>
<body>
    {{template "nav" .}}
    <main>{{block "content" .}}{{end}}</main>
    {{template "footer" .}}
    <script src="/static/js/app.js" defer></script>
</body>
</html>
```

**Handler pattern:**
```go
func (h *PageHandler) ComicDetail(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    comic, err := h.comicSvc.GetByID(r.Context(), id)
    if err != nil {
        h.renderError(w, r, http.StatusNotFound, "Comic not found")
        return
    }
    h.render(w, r, "comic.html", map[string]any{
        "Comic":   comic,
        "User":    auth.UserFromCtx(r.Context()),
    })
}
```

---

## 4. API Contract Reference

### Authentication

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | ❌ | Email + password login |
| `POST` | `/api/v1/auth/logout` | ✅ | Revoke current session |
| `POST` | `/api/v1/auth/refresh` | ❌ | Refresh access token using cookie |
| `POST` | `/api/v1/auth/register` | ❌ | Create new account |

### Comics

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/comics` | ❌ | List/filter comics. Query: `status`, `tags`, `demo`, `rating`, `sort`, `page`, `limit` |
| `GET` | `/api/v1/comics/{id}` | ❌ | Comic detail + chapter list |
| `GET` | `/api/v1/comics/{id}/chapters` | ❌ | Paginated chapter list with language filter |
| `GET` | `/api/v1/comics/{id}/art` | ❌ | Art gallery |
| `GET` | `/api/v1/comics/{id}/recommendations` | ❌ | Recommendations sorted by upvotes |
| `POST` | `/api/v1/comics/{id}/follow` | ✅ | Add to library |
| `DELETE` | `/api/v1/comics/{id}/follow` | ✅ | Remove from library |

### Library

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/me/library` | ✅ | User's library. Filter: `status` |
| `GET` | `/api/v1/me/library/history` | ✅ | Recently viewed comics |
| `PUT` | `/api/v1/me/library/{comicId}` | ✅ | Update shelf entry (status, score, notes) |
| `GET` | `/api/v1/me/lists` | ✅ | Custom lists |
| `GET` | `/api/v1/me/feed` | ✅ | Activity feed |

### Chapters

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/chapters/{id}` | ❌ | Chapter detail + page list |
| `POST` | `/api/v1/chapters/{id}/read` | ✅ | Mark chapter as read |

---

## 5. Error Handling

All errors are returned as:
```json
{
  "error": "Human readable message",
  "code": "MACHINE_READABLE_CODE"
}
```

| HTTP Status | Code | Meaning |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Request body/params failed validation |
| `401` | `UNAUTHORIZED` | Missing or invalid auth token |
| `403` | `FORBIDDEN` | Authenticated but insufficient role |
| `404` | `NOT_FOUND` | Resource does not exist (or soft-deleted) |
| `409` | `CONFLICT` | Duplicate (unique constraint) |
| `429` | `RATE_LIMITED` | Too many requests |
| `500` | `INTERNAL_ERROR` | Unexpected server error (never expose details) |

**Rule:** Never return Go error strings or stack traces in API responses. Log them server-side with correlation ID.

---

## 6. Background Workers

All background workers run as goroutines started in `main.go`:

```go
// worker/counter_flush.go
// Flushes Redis INCR counters → PostgreSQL every 60 seconds
func RunCounterFlush(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) {
    ticker := time.NewTicker(60 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            flushViewCounts(ctx, pool, rdb)
        }
    }
}
```

| Worker | Interval | Purpose |
|---|---|---|
| `counter_flush` | 60s | Redis `comic:{id}:views` → `core.comic.viewcount` |
| `hasnew_updater` | On crawler completion | Set `library.entry.hasnew = TRUE` for followers |
| `bayesian_rating` | 15 min | Recompute `comic.ratingbayesian` for changed comics |
| `partition_creator` | Monthly (1st of month) | Pre-create next partition for `pageview`, `chaptersession`, `crawler.log` |
| `viewhistory_trim` | Daily | Delete rows beyond 500 per user in `library.viewhistory` |
| `session_cleanup` | Hourly | Delete expired + revoked sessions from `users.session` |
