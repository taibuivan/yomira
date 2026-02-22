# Go Coding Standards — Yomira Backend

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Status:** Active — mandatory for all Go code in `src/`  
> **Applies to:** All packages under `src/cmd/`, `src/internal/`, `src/pkg/`, `src/common/`

---

## Table of Contents

1. [Guiding Principles](#1-guiding-principles)
2. [Project Layout Rules](#2-project-layout-rules)
3. [Naming Conventions](#3-naming-conventions)
4. [Code Formatting](#4-code-formatting)
5. [Comment & Documentation Standards](#5-comment--documentation-standards)
6. [Package Design Rules](#6-package-design-rules)
7. [Function & Method Rules](#7-function--method-rules)
8. [Error Handling Rules](#8-error-handling-rules)
9. [Concurrency Rules](#9-concurrency-rules)
10. [Dependency Injection Rules](#10-dependency-injection-rules)
11. [Layer Contracts](#11-layer-contracts)
12. [Testing Standards](#12-testing-standards)
13. [Logging Standards](#13-logging-standards)
14. [Security Rules](#14-security-rules)
15. [Performance Rules](#15-performance-rules)
16. [Anti-Patterns Catalogue](#16-anti-patterns-catalogue)
17. [Linter Configuration](#17-linter-configuration)

---

## 1. Guiding Principles

These are the non-negotiable values every line of code must respect.

| # | Principle | Meaning |
|---|---|---|
| 1 | **Explicitness over magic** | Avoid interface{}, reflection tricks, and hidden globals. Prefer explicit types and injection. |
| 2 | **Errors are values** | Never ignore, swallow, or `log + return` the same error twice. Treat errors as deliberate return values. |
| 3 | **Package = responsibility** | A package owns a single coherent responsibility. Not a category of types. |
| 4 | **Dependencies flow inward** | `api` → `service` → `domain`. Never upward. `domain` depends on nothing except the standard library. |
| 5 | **Zero globals (except loggers)** | No `var db *pgxpool.Pool` at package level. All state is injected via constructors. |
| 6 | **Tests are first-class citizens** | Untested behaviour does not exist. Every exported function has at least one test. |
| 7 | **Context carries the request** | `context.Context` is the first parameter of every function that touches I/O. Never store context in a struct. |
| 8 | **Simplicity over cleverness** | If a solution requires a comment to explain *how* it works, simplify the code instead. |

---

## 2. Project Layout Rules

### Canonical directory map

```
src/
├── cmd/
│   └── api/
│       └── main.go              # Wiring only — no business logic
│
├── internal/
│   ├── config/                  # Config struct, env parsing (Read once at startup)
│   ├── domain/                  # Pure Go structs + repository interfaces. Zero imports.
│   ├── service/                 # Business logic. Depends on domain interfaces only.
│   ├── storage/
│   │   ├── postgres/            # Implements domain repository interfaces (SQL, pgx)
│   │   └── redis/               # Implements session/counter stores
│   ├── auth/                    # JWT sign/verify, middleware
│   ├── api/                     # HTTP handlers, server.go, response helpers
│   ├── worker/                  # Background goroutines (counter flush, partitions)
│   └── crawler/                 # Source plugins, scheduler
│
├── pkg/                         # Reusable, non-internal packages (safe to import externally)
│   ├── pagination/
│   ├── slug/
│   └── uuidv7/
│
└── common/
    └── DML/                     # SQL schema source of truth
```

### Rules

| Rule | Rationale |
|---|---|
| `internal/` prevents external import of application internals | Go enforces this at compile time |
| `pkg/` is for truly reusable packages with no app-specific logic | If it touches `domain`, it belongs in `internal` |
| `cmd/api/main.go` is a wiring file only — ≤ 150 lines | Prevents "God main" anti-pattern |
| No `utils/`, `helpers/`, `common/` inside `internal/` | Functions belong to the package that uses them |
| Shared infrastructure lives in `internal/shared/` (apperr, validate, respond) | Not `utils` — it has a named responsibility |

---

## 3. Naming Conventions

### 3.1 Variables and constants

```go
// ✅ Correct — camelCase, descriptive
comicID    := chi.URLParam(r, "id")
pageLimit  := 20
retryCount := 3

// ✅ Correct — acronyms: all-caps when standalone, capitalised when embedded
userID   string
comicURL string
jsonBody []byte
httpClient *http.Client

// ❌ Wrong — too short, unclear
n   := 20
cnt := 3
id  := chi.URLParam(r, "id")   // ambiguous — what kind of ID?
```

### 3.2 Constants

```go
// ✅ Typed constants with iota — use when order matters
type ComicStatus string
const (
    ComicStatusOngoing   ComicStatus = "ongoing"
    ComicStatusCompleted ComicStatus = "completed"
    ComicStatusHiatus    ComicStatus = "hiatus"
)

// ✅ Untyped constants for limits/timeouts
const (
    maxPageLimit       = 100
    defaultPageLimit   = 20
    sessionTTL         = 30 * 24 * time.Hour
    accessTokenTTL     = 15 * time.Minute
)

// ❌ Wrong — magic numbers inline
rows, err := db.Query(ctx, q, 20)    // what is 20?
time.Sleep(60 * time.Second)         // use named constant
```

### 3.3 Functions and methods

```go
// ✅ Verb-noun pattern for actions
func (s *ComicService) GetByID(ctx context.Context, id string) (*domain.Comic, error)
func (s *ComicService) ListByTag(ctx context.Context, tag string, p Pagination) ([]*domain.Comic, int, error)
func (r *comicRepo) SoftDelete(ctx context.Context, id string) error

// ✅ Noun pattern for constructors
func NewComicService(repo domain.ComicRepository, cache *redis.Client) *ComicService
func NewComicHandler(svc ComicService) *ComicHandler

// ✅ Boolean functions start with Is/Has/Can/Should
func (c *Comic) IsCompleted() bool
func (u *AuthUser) HasRole(role string) bool
func (v *Validator) HasErrors() bool

// ❌ Wrong — vague verbs
func (s *ComicService) Process(...)  // process what?
func (r *comicRepo) Handle(...)      // handle what?
func DoStuff(...)                    // never acceptable
```

### 3.4 Types and interfaces

```go
// ✅ Interfaces: describe behaviour, often -er suffix
type ComicRepository interface { ... }   // not IComicRepository or ComicRepo
type SessionStore    interface { ... }
type EventPublisher  interface { ... }

// ✅ Structs: noun, no suffix
type Comic   struct { ... }
type Chapter struct { ... }
type AuthUser struct { ... }    // not UserAuth or AuthUserStruct

// ✅ Request/response types: named by purpose
type CreateComicRequest  struct { ... }
type ComicDetailResponse struct { ... }
type PaginatedResponse[T any] struct {
    Data  []T `json:"data"`
    Meta  PaginationMeta `json:"meta"`
}

// ❌ Wrong suffixes
type IComicRepository interface {}   // Java-style prefix
type ComicDTO         struct {}      // DTO suffix not idiomatic Go
type ComicVO          struct {}      // VO suffix not idiomatic Go
```

### 3.5 Packages

```go
// ✅ Short, lowercase, no underscores, no plurals
package comic      // not comics, not comic_service
package storage    // not storages
package auth       // not authentication
package pagination // descriptive noun is fine

// ✅ Test files: _test suffix, same package OR external package
package comic        // white-box test (access unexported)
package comic_test   // black-box test (only exported API)

// ❌ Wrong
package Comic         // never uppercase
package comic_service // no underscores
package utils         // not a responsibility — split into specific packages
```

### 3.6 Files

| Pattern | Usage |
|---|---|
| `comic.go` | Primary type definitions and constructors for a type |
| `comic_repo.go` | Repository implementation for comic |
| `comic_handler.go` | HTTP handlers for comic domain |
| `comic_service.go` | Service business logic for comic |
| `errors.go` | Sentinel errors for the package |
| `mock_repo_test.go` | Test mocks — `_test.go` suffix keeps them test-only |
| `server.go`, `router.go` | Router setup and middleware wiring |
| `doc.go` | Package-level documentation comment |

---

## 4. Code Formatting

### 4.1 Tooling (mandatory, enforced in CI)

| Tool | Rule |
|---|---|
| `gofmt` | All code must be `gofmt`-formatted before commit. Zero tolerance. |
| `goimports` | Manage import groups automatically. Run on save. |
| `golangci-lint` | Full lint suite — zero new warnings on every PR. |

### 4.2 Import grouping

Always three groups, separated by blank lines. `goimports` handles this automatically.

```go
import (
    // 1. Standard library
    "context"
    "errors"
    "fmt"
    "net/http"

    // 2. Third-party packages
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "golang.org/x/crypto/bcrypt"

    // 3. Internal packages (module path prefix)
    "github.com/buivan/yomira/internal/domain"
    "github.com/buivan/yomira/internal/shared/apperr"
    "github.com/buivan/yomira/internal/shared/respond"
)
```

### 4.3 Line length and wrapping

```go
// ✅ Wrap function calls — align arguments when span > 100 chars
err := s.repo.Create(
    ctx,
    &domain.Comic{
        ID:     uuidv7.New().String(),
        Title:  req.Title,
        Slug:   slug.From(req.Title),
        Status: domain.ComicStatusOngoing,
    },
)

// ✅ Multi-return: declare vars on separate lines when 3+
comicID, chapterID, pageIndex, err := parseReadParams(r)

// ❌ Wrong — trailing comma missing on multi-line struct/call
comic := domain.Comic{
    ID:    "...",
    Title: "Test"     // ← missing comma causes error on next reformat
}
```

### 4.4 Blank lines

```go
// ✅ One blank line between logical sections of a function
func (s *ComicService) Register(ctx context.Context, req CreateComicRequest) (*domain.Comic, error) {
    // 1. Validate
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // 2. Slug uniqueness check
    if _, err := s.repo.FindBySlug(ctx, slug.From(req.Title)); err == nil {
        return nil, apperr.Conflict("A comic with this title already exists")
    }

    // 3. Persist
    comic := &domain.Comic{
        ID:    uuidv7.New().String(),
        Title: req.Title,
        Slug:  slug.From(req.Title),
    }
    if err := s.repo.Create(ctx, comic); err != nil {
        return nil, err
    }

    return comic, nil
}
```

---

## 5. Comment & Documentation Standards

### 5.1 Package doc comments

Every package **must** have a `doc.go` or a package comment in the primary file.

```go
// Package comic provides the business logic for managing comics,
// chapters, and their associated metadata.
//
// The ComicService orchestrates reads and writes via the domain.ComicRepository
// interface. It performs validation and slug uniqueness enforcement before
// delegating to storage.
package comic
```

### 5.2 Exported symbol comments

Every exported function, type, constant, and variable **must** have a godoc comment.
The comment must begin with the symbol name and describe *what* it does, not *how*.

```go
// Comic represents a serialised publication (manga, manhwa, webtoon, etc.)
// in the Yomira catalogue. It is the central aggregate of the core domain.
type Comic struct { ... }

// GetByID returns the comic with the given ID.
// It returns apperr.ErrNotFound if the comic does not exist or is soft-deleted.
func (s *ComicService) GetByID(ctx context.Context, id string) (*domain.Comic, error) { ... }

// ComicStatusOngoing indicates the publication is actively updating.
const ComicStatusOngoing ComicStatus = "ongoing"
```

### 5.3 Inline comments — rules

```go
// ✅ Comment explains WHY, not WHAT
// Bcrypt with cost 12 — sufficient for 2026 hardware while staying under 250ms/hash
hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// ✅ Mark intentional non-obvious decisions
// Intentionally ignore error here — Expire is best-effort; key will expire naturally.
_ = rdb.Expire(ctx, key, sessionTTL)

// ✅ TODO with owner and ticket reference
// TODO(tai): Replace with Redis pub/sub once #88 is done.

// ❌ Wrong — comment restates the code
// Increment i by 1
i++

// ❌ Wrong — commented-out code (use git, not comments)
// comic.Title = strings.ToLower(comic.Title)
```

### 5.4 File header

All non-generated Go files start with a brief attribution block:

```go
// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com
```

Generated files (mocks, protobuf, sqlc) must have:

```go
// Code generated by mockery v2. DO NOT EDIT.
```

---

## 6. Package Design Rules

### 6.1 Dependency direction (strict)

```
cmd/api/main.go
    │
    ├── internal/api/         (HTTP handlers — knows service interfaces)
    │       │
    │       └── internal/service/   (business logic — knows domain interfaces)
    │               │
    │               └── internal/domain/    (pure structs + repository interfaces)
    │                                               ▲
    │               ┌───────────────────────────────┘
    └── internal/storage/     (implements domain interfaces — knows domain structs)
```

- `domain` imports NOTHING from the project
- `service` imports ONLY `domain` and `shared/`
- `storage` imports ONLY `domain` and `shared/`
- `api` imports ONLY `service` interfaces and `shared/`
- **Horizontal imports are forbidden** — `storage` cannot import `service`, `api` cannot import `storage`

### 6.2 Package size

| Metric | Limit | Action if exceeded |
|---|---|---|
| Files per package | ≤ 15 | Split into sub-packages by sub-domain |
| Lines per file | ≤ 500 | Extract helpers into separate file |
| Exported functions per package | ≤ 20 | Refactor — package is doing too much |

### 6.3 Circular imports

Go will refuse to compile circular imports. If you find yourself needing one:
1. Extract the shared type into `internal/domain` (domain structs)
2. Extract the shared behaviour into `internal/shared/` (utility logic)
3. Never introduce an intermediate package just to break a cycle — fix the design.

---

## 7. Function & Method Rules

### 7.1 Function length

| Size | Rule |
|---|---|
| ≤ 30 lines | Preferred |
| 31–60 lines | Acceptable with clear section comments |
| > 60 lines | **Mandatory refactor** — extract helpers |

### 7.2 Parameter count

```go
// ✅ ≤ 4 parameters — acceptable
func CreateChapter(ctx context.Context, comicID, title string, number float64) error

// ✅ > 4 parameters — use a request struct
type CreateChapterRequest struct {
    ComicID      string
    Title        string
    Number       float64
    Language     string
    ExternalURL  string
    Translators  []string
}
func CreateChapter(ctx context.Context, req CreateChapterRequest) error

// ❌ Wrong — parameter explosion
func CreateChapter(ctx context.Context, comicID, title, language, externalURL string,
    number float64, translators []string, isLocked bool) error
```

### 7.3 Return values

```go
// ✅ Return (value, error) — the Go canonical pair
func (r *comicRepo) FindByID(ctx context.Context, id string) (*domain.Comic, error)

// ✅ Multiple returns when semantically clear
func ListComics(ctx context.Context, f Filter) ([]*domain.Comic, int, error)
//                                                  data       total  err

// ✅ Named returns ONLY for documentation, never for naked returns
func divide(a, b float64) (result float64, err error) {
    if b == 0 {
        return 0, errors.New("division by zero")   // explicit return, not naked
    }
    return a / b, nil
}

// ❌ Wrong — naked return in non-trivial function
func parseComic(s string) (id string, title string, err error) {
    // ... 30 lines of logic ...
    return   // ← naked return — where do id and title come from?
}
```

### 7.4 Receiver naming

```go
// ✅ Short, consistent, matches type abbreviation
func (r *comicRepo) FindByID(...)    // r = repo
func (s *ComicService) GetByID(...) // s = service
func (h *ComicHandler) GetByID(...) // h = handler
func (c *Comic) IsCompleted() bool  // c = comic

// ❌ Wrong — long or generic receivers
func (self *comicRepo) FindByID(...)  // no 'self'
func (this *comicRepo) FindByID(...) // no 'this'
func (comicRepo *comicRepo) FindByID(...) // repeats type name
```

### 7.5 Early returns (guard clauses)

```go
// ✅ Fail fast — happy path has minimal indentation
func (h *ComicHandler) Create(w http.ResponseWriter, r *http.Request) {
    user := auth.MustGetUser(r.Context())
    if user.Role != "admin" {
        respond.Error(w, apperr.Forbidden("Admin only"))
        return
    }

    var req CreateComicRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respond.Error(w, apperr.ValidationError("Invalid request body"))
        return
    }

    comic, err := h.svc.Create(r.Context(), req)
    if err != nil {
        respond.Error(w, err)
        return
    }

    respond.Created(w, comic)
}

// ❌ Wrong — pyramid of doom
func (h *ComicHandler) Create(w http.ResponseWriter, r *http.Request) {
    user := auth.MustGetUser(r.Context())
    if user.Role == "admin" {
        var req CreateComicRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
            comic, err := h.svc.Create(r.Context(), req)
            if err == nil {
                respond.Created(w, comic)
            } else {
                respond.Error(w, err)
            }
        } else {
            respond.Error(w, apperr.ValidationError("Invalid request body"))
        }
    } else {
        respond.Error(w, apperr.Forbidden("Admin only"))
    }
}
```

---

## 8. Error Handling Rules

> Full reference: [ERROR_HANDLING.md](./ERROR_HANDLING.md)

### Quick reference

| Rule | Example |
|---|---|
| Return sentinel errors from storage | `return nil, ErrComicNotFound` |
| Wrap unexpected errors with context | `return nil, fmt.Errorf("ComicRepo.Create: %w", err)` |
| Do not re-wrap `AppError` in service | `if apperr.IsAppError(err) { return nil, err }` |
| Log once at origin, not at every layer | Storage logs DB errors; handler logs 5xx via `RespondError` |
| Never expose raw `pgx`/`sql` errors | `dberr.Translate(err, ErrComicNotFound)` |
| Never expose internal strings to client | Use `AppError.Message`, never `err.Error()` |
| Always check `errors.Is` / `errors.As` | Never string-match error messages |

### Error propagation flow

```
storage layer       → returns sentinel AppError OR fmt.Errorf("Context: %w", rawErr)
service layer       → passes AppError through; wraps unexpected as apperr.Internal(...)
handler layer       → calls respond.Error(w, r, err) — only place that writes HTTP
middleware layer    → calls WriteError directly (no service involved)
```

---

## 9. Concurrency Rules

### 9.1 Context propagation

```go
// ✅ Always pass ctx as first argument — no exceptions
func (r *comicRepo) FindByID(ctx context.Context, id string) (*domain.Comic, error) {
    row := r.pool.QueryRow(ctx, query, id)
    ...
}

// ❌ Wrong — context stored in struct
type comicRepo struct {
    pool *pgxpool.Pool
    ctx  context.Context   // ← never store context
}
```

### 9.2 Goroutine lifecycle

```go
// ✅ Every goroutine must have a clear termination condition
func RunCounterFlush(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()    // ← always defer Stop() to prevent goroutine leak

    for {
        select {
        case <-ctx.Done():
            return           // ← clean shutdown
        case <-ticker.C:
            flushViewCounts(ctx, pool, rdb)
        }
    }
}

// ✅ Use WaitGroup for goroutine groups with fan-out
var wg sync.WaitGroup
for _, job := range jobs {
    wg.Add(1)
    go func(j Job) {
        defer wg.Done()
        processJob(ctx, j)
    }(job)
}
wg.Wait()

// ❌ Wrong — fire-and-forget goroutine with no lifecycle management
go func() {
    flushViewCounts(pool, rdb)   // when does this stop? what if it panics?
}()
```

### 9.3 Shared state

```go
// ✅ Protect shared state with mutex — document what the mutex protects
type Settings struct {
    mu   sync.RWMutex    // protects values
    values map[string]string
}

func (s *Settings) Get(key string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.values[key]
}

// ✅ Prefer channels for signalling, mutexes for protecting data
// ✅ Use sync/atomic for simple counters
var viewCount atomic.Int64
viewCount.Add(1)

// ❌ Wrong — no synchronisation on shared map
var cache = map[string]*Comic{}
// concurrent reads+writes → data race
```

### 9.4 Channel sizing

```go
// ✅ Unbuffered: synchronous handoff (default, prefer this)
results := make(chan Result)

// ✅ Buffered: only when you have a concrete reason and documented capacity
// Buffer = crawlerWorkerCount ensures producers never block on slow consumers
jobs := make(chan Job, crawlerWorkerCount)

// ❌ Wrong — arbitrarily buffered channels
results := make(chan Result, 999)  // why 999?
```

---

## 10. Dependency Injection Rules

Yomira uses **manual constructor injection** — no DI framework, no service locator.

### 10.1 Constructor pattern

```go
// ✅ Constructor receives all dependencies as explicit parameters
type ComicService struct {
    repo   domain.ComicRepository    // interface, not concrete type
    cache  *redis.Client
    logger *slog.Logger
}

func NewComicService(
    repo   domain.ComicRepository,
    cache  *redis.Client,
    logger *slog.Logger,
) *ComicService {
    return &ComicService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}

// ❌ Wrong — global variable used as dependency
var globalDB *pgxpool.Pool

func (s *ComicService) GetByID(ctx context.Context, id string) (*domain.Comic, error) {
    row := globalDB.QueryRow(ctx, q, id)   // ← hidden dependency, untestable
    ...
}
```

### 10.2 Wiring in main.go

```go
// cmd/api/main.go — wiring only. No logic.
func main() {
    cfg, err := config.Load()
    must(err, "config")

    pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
    must(err, "db pool")

    rdb, err := storage.NewRedis(ctx, cfg.RedisURL)
    must(err, "redis")

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Storage layer
    comicRepo   := postgres.NewComicRepo(pool)
    userRepo    := postgres.NewUserRepo(pool)

    // Service layer
    comicSvc  := service.NewComicService(comicRepo, rdb, logger)
    authSvc   := service.NewAuthService(userRepo, rdb, logger, cfg)

    // API layer
    comicHandler := api.NewComicHandler(comicSvc)
    authHandler  := api.NewAuthHandler(authSvc)

    server := api.NewServer(cfg, logger, comicHandler, authHandler)
    server.ListenAndServe()
}
```

### 10.3 Interface rule

- Define interfaces **where they are consumed**, not where they are implemented.
- `domain.ComicRepository` is defined in `internal/domain/` because service consumes it.
- `postgres.comicRepo` struct implements that interface in `internal/storage/postgres/`.

```go
// ✅ Correct — interface in domain, consumed by service
// internal/domain/comic.go
type ComicRepository interface {
    FindByID(ctx context.Context, id string) (*Comic, error)
    List(ctx context.Context, f ComicFilter) ([]*Comic, int, error)
    Create(ctx context.Context, c *Comic) error
}

// internal/storage/postgres/comic_repo.go
// compile-time assertion — catches interface drift early
var _ domain.ComicRepository = (*comicRepo)(nil)

type comicRepo struct { pool *pgxpool.Pool }

func (r *comicRepo) FindByID(ctx context.Context, id string) (*domain.Comic, error) { ... }
```

---

## 11. Layer Contracts

### 11.1 `domain/` — zero dependencies

```go
// ALLOWED: standard library only
import (
    "context"
    "time"
)

// FORBIDDEN: any external package, any internal package
import "github.com/jackc/pgx/v5"          // ❌ storage leak into domain
import "github.com/buivan/yomira/internal/shared/apperr" // ❌ even apperr
```

### 11.2 `service/` — business logic only

```go
// ALLOWED
import (
    "github.com/buivan/yomira/internal/domain"
    "github.com/buivan/yomira/internal/shared/apperr"
    "github.com/buivan/yomira/internal/shared/validate"
)

// FORBIDDEN
import "github.com/jackc/pgx/v5"          // ❌ storage concern
import "net/http"                          // ❌ transport concern
```

### 11.3 `storage/` — SQL and external stores

```go
// ALLOWED
import (
    "github.com/buivan/yomira/internal/domain"
    "github.com/buivan/yomira/internal/shared/apperr"
    "github.com/buivan/yomira/internal/shared/dberr"
    "github.com/jackc/pgx/v5/pgxpool"
)

// FORBIDDEN
import "github.com/buivan/yomira/internal/service"  // ❌ upward dependency
import "net/http"                                    // ❌ transport concern
```

### 11.4 `api/` (handlers) — HTTP only

```go
// ALLOWED
import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/buivan/yomira/internal/domain"
    "github.com/buivan/yomira/internal/shared/respond"
    "github.com/buivan/yomira/internal/shared/apperr"
)

// FORBIDDEN
import "github.com/jackc/pgx/v5"                    // ❌ bypass service
import "github.com/buivan/yomira/internal/storage"  // ❌ bypass service
```

### 11.5 What lives where: quick lookup

| Thing | Where |
|---|---|
| `Comic`, `Chapter`, `User` structs | `internal/domain/` |
| `ComicRepository` interface | `internal/domain/` (consumed by service) |
| `ComicStatus` const | `internal/domain/comic.go` |
| SQL queries | `internal/storage/postgres/` |
| Bcrypt password hash | `internal/service/auth.go` |
| JWT sign/verify | `internal/auth/token.go` |
| HTTP middleware | `internal/api/middleware/` |
| JSON response helpers | `internal/shared/respond/` |
| Error types | `internal/shared/apperr/` |
| Sentinel errors | `internal/storage/<pkg>/errors.go` |
| DB error translation | `internal/shared/dberr/` |
| Slug generation | `internal/pkg/slug/` |
| UUIDv7 generation | `internal/pkg/uuidv7/` |

---

## 12. Testing Standards

> Full reference: [TESTING.md](./TESTING.md)

### 12.1 Mandatory test coverage

| Layer | Minimum coverage | Priority |
|---|---|---|
| `shared/` (apperr, validate, respond) | ≥ 90% | Highest — pure functions |
| `middleware/` | ≥ 85% | All error paths |
| `service/` | ≥ 75% | Core business logic |
| `storage/` (integration) | ≥ 65% | SQL correctness |
| `api/` (handlers) | ≥ 60% | HTTP contracts |

### 12.2 Table-driven tests (mandatory pattern)

```go
// ✅ All test functions use table-driven pattern
func TestComicService_GetByID(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        mockFn  func(m *mockComicRepo)
        wantErr error
    }{
        {
            name: "found",
            id:   "01952fa3-0000-7000-0000-000000000001",
            mockFn: func(m *mockComicRepo) {
                m.On("FindByID", mock.Anything, mock.Anything).
                    Return(&domain.Comic{Title: "Solo Leveling"}, nil)
            },
        },
        {
            name: "not found",
            id:   "00000000-0000-0000-0000-000000000000",
            mockFn: func(m *mockComicRepo) {
                m.On("FindByID", mock.Anything, mock.Anything).
                    Return(nil, storage_comic.ErrComicNotFound)
            },
            wantErr: storage_comic.ErrComicNotFound,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            repo := &mockComicRepo{}
            tc.mockFn(repo)
            svc := NewComicService(repo, nil, slog.Default())

            _, err := svc.GetByID(t.Context(), tc.id)   // t.Context() — Go 1.24+

            assert.ErrorIs(t, err, tc.wantErr)
            repo.AssertExpectations(t)
        })
    }
}
```

### 12.3 Test file rules

| Rule | Rationale |
|---|---|
| Test files in the same package as the code | Whitebox access to unexported symbols |
| `_test.go` suffix on all test files | Go toolchain excludes from production build |
| `mock_*_test.go` naming for mock types | Clearly separates mocks from production code |
| `t.Helper()` on all helper functions | Correct line numbers in failure output |
| Never `t.Skip()` in CI-running tests | All tests must pass in CI |
| Use `require.NoError` for setup, `assert` for assertions | Setup failures should stop the test immediately |

### 12.4 No test pollution

```go
// ✅ Clean state between tests using t.Cleanup
func TestSomething(t *testing.T) {
    db := testdb.Setup(t)
    t.Cleanup(func() {
        db.Exec("TRUNCATE core.comic CASCADE")
    })
    ...
}

// ✅ Use subtests to isolate state
t.Run("scenario A", func(t *testing.T) { ... })
t.Run("scenario B", func(t *testing.T) { ... })
```

---

## 13. Logging Standards

Yomira uses `log/slog` (stdlib, Go 1.21+). No `fmt.Println`, no third-party logger.

### 13.1 Log levels

| Level | When to use |
|---|---|
| `Debug` | Development only. Gates on `cfg.Debug = true`. Never in production by default. |
| `Info` | Normal operation events (server started, job completed, 2xx/3xx requests). |
| `Warn` | Expected but notable conditions (4xx errors, rate limit hits, auth failures). |
| `Error` | Unexpected failures (5xx errors, DB errors, panics). |

### 13.2 Structured fields (mandatory)

```go
// ✅ Always use structured fields — never interpolate into the message string
logger.ErrorContext(ctx, "comic not found in storage",
    slog.String("comic_id", id),
    slog.String("request_id", middleware.GetRequestID(ctx)),
    slog.Any("error", err),
)

// ❌ Wrong — unstructured log message
logger.Error(fmt.Sprintf("comic %s not found: %v", id, err))
```

### 13.3 Mandatory fields by layer

| Layer | Mandatory slog fields |
|---|---|
| Middleware (request log) | `method`, `path`, `status`, `latency_ms`, `request_id`, `ip` |
| Auth middleware | `reason`, `ip`, `request_id` |
| Storage errors | `error`, `query context` (e.g. `comic_id`) |
| Service | `request_id` via context (extracted by logger middleware) |
| Panic recovery | `panic`, `stack`, `request_id` |

### 13.4 Log once, not many

```go
// ✅ Log at origin, return upward — never log the same error twice
func (r *comicRepo) Create(ctx context.Context, c *domain.Comic) error {
    _, err := r.pool.Exec(ctx, insertQuery, c.ID, c.Title, c.Slug)
    if err != nil {
        slog.ErrorContext(ctx, "ComicRepo.Create failed",
            slog.String("comic_id", c.ID),
            slog.Any("error", err))
        return dberr.Translate(err)
    }
    return nil
}

// ❌ Wrong — log at every layer
func (s *ComicService) Create(ctx context.Context, req CreateComicRequest) (*domain.Comic, error) {
    comic, err := s.repo.Create(ctx, ...)
    if err != nil {
        slog.Error("Create failed", "error", err)   // ← duplicate log
        return nil, err
    }
    ...
}
```

---

## 14. Security Rules

> Full reference: [SECURITY.md](./SECURITY.md)

| Rule | Implementation |
|---|---|
| Never log sensitive data | No passwords, tokens, or PII in log fields. Hash or redact before logging. |
| Always hash passwords with bcrypt | `bcrypt.GenerateFromPassword([]byte(pw), 12)` — cost ≥ 12 |
| Store only SHA-256(refresh_token) in DB | Raw token lives only in the cookie |
| parameterized queries only | `pool.Exec(ctx, "SELECT ... WHERE id = $1", id)` — never format SQL with user input |
| Validate all user input at service layer | Use `validate.Validator` before any DB write |
| Sanitise slugs — never trust client-provided slugs | `slug.From(req.Title)` — compute server-side |
| Rate-limit all auth endpoints | `RateLimiter(redis, RateLimit{Requests: 10, Window: 15*time.Minute, KeyFunc: ByIP})` |
| JWT RS256 only — never HS256 | Key loaded from file at startup, not from env string |
| No secrets in source code | All secrets via `.env` — never in `config.go` defaults |
| Set `HttpOnly; SameSite=Strict` on refresh token cookie | Prevents XSS token theft |

---

## 15. Performance Rules

### 15.1 Database

```go
// ✅ Use pgxpool — never single connection
pool, _ := pgxpool.New(ctx, cfg.DatabaseURL)

// ✅ Use transactions for multi-step writes
tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
defer tx.Rollback(ctx)
// ... multiple writes ...
tx.Commit(ctx)

// ✅ Use RETURNING to avoid second SELECT
const q = `
    INSERT INTO core.comic (id, title, slug, status)
    VALUES ($1, $2, $3, $4)
    RETURNING id, createdat`

// ✅ Batch inserts with pgx CopyFrom for bulk operations (crawler)
_, err = pool.CopyFrom(ctx,
    pgx.Identifier{"crawler", "page"},
    []string{"id", "chapter_id", "page_number", "image_url"},
    pgx.CopyFromRows(rows),
)
```

### 15.2 Memory

```go
// ✅ Pre-allocate slices when size is known
comics := make([]*domain.Comic, 0, len(rows))

// ✅ Reuse buffers with sync.Pool for hot paths
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

// ✅ Stream large responses — don't materialise entire result set
rows, err := pool.Query(ctx, q)
defer rows.Close()
enc := json.NewEncoder(w)
for rows.Next() {
    var c domain.Comic
    rows.Scan(&c.ID, &c.Title)
    enc.Encode(c)
}
```

### 15.3 HTTP

```go
// ✅ Always set read/write timeouts on the server
server := &http.Server{
    Addr:         ":" + cfg.ServerPort,
    Handler:      router,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,
}

// ✅ Decode request body with size limit
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)  // 1 MiB max body
```

---

## 16. Anti-Patterns Catalogue

These patterns are **rejected in code review** without exception.

### 16.1 Global mutable state

```go
// ❌ Never
var db *pgxpool.Pool

// ✅ Always inject
type comicRepo struct { pool *pgxpool.Pool }
```

### 16.2 `init()` for side effects

```go
// ❌ init() with side effects is invisible and untestable
func init() {
    db, _ = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
}

// ✅ Explicit setup in main.go
pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
must(err, "db pool")
```

### 16.3 Empty interface as escape hatch

```go
// ❌ Lazy generic typing
func Process(data interface{}) interface{} { ... }

// ✅ Use generics (Go 1.21+) or specific types
func Map[T, U any](slice []T, fn func(T) U) []U { ... }
```

### 16.4 Panic in business logic

```go
// ❌ Never panic on expected conditions
func GetComic(id string) *Comic {
    c, err := repo.FindByID(ctx, id)
    if err != nil {
        panic(err)   // kills the server process
    }
    return c
}

// ✅ Return errors
func GetComic(ctx context.Context, id string) (*Comic, error) {
    return repo.FindByID(ctx, id)
}
```

### 16.5 Ignored errors

```go
// ❌ Silently discard errors
comic, _ := repo.FindByID(ctx, id)

// ✅ Handle every error
comic, err := repo.FindByID(ctx, id)
if err != nil {
    return nil, err
}
```

Exceptions — the only acceptable `_` error discards:
```go
// Best-effort operations — explicitly documented
_ = rdb.Expire(ctx, key, sessionTTL)              // Redis Expire is best-effort
_ = json.NewEncoder(w).Encode(body)               // write after WriteHeader — can't undo
defer rows.Close()                                 // panic would propagate anyway
```

### 16.6 ORM / query magic

```go
// ❌ ORM hides SQL — no visibility into what queries are running
db.Where("status = ?", "ongoing").Find(&comics)

// ✅ Explicit SQL — readable, optimisable, indexable
const q = `
    SELECT id, title, status, viewcount
    FROM core.comic
    WHERE status = $1
      AND deletedat IS NULL
    ORDER BY viewcount DESC
    LIMIT $2 OFFSET $3`
```

### 16.7 Handler doing business logic

```go
// ❌ Handler contains domain logic
func (h *ComicHandler) Follow(w http.ResponseWriter, r *http.Request) {
    comicID := chi.URLParam(r, "id")
    user    := auth.MustGetUser(r.Context())

    // ❌ SQL in handler
    _, err := h.db.Exec(ctx, `INSERT INTO library.entry ...`, user.ID, comicID)
    if err != nil {
        respond.Error(w, err)
        return
    }
    respond.NoContent(w)
}

// ✅ Handler delegates to service
func (h *ComicHandler) Follow(w http.ResponseWriter, r *http.Request) {
    comicID := chi.URLParam(r, "id")
    user    := auth.MustGetUser(r.Context())

    if err := h.libSvc.Follow(r.Context(), user.ID, comicID); err != nil {
        respond.Error(w, err)
        return
    }
    respond.NoContent(w)
}
```

### 16.8 `time.Sleep` in production code

```go
// ❌ Blocks goroutine, ignores context cancellation
time.Sleep(5 * time.Second)

// ✅ Respect context cancellation
select {
case <-time.After(5 * time.Second):
    doSomething()
case <-ctx.Done():
    return ctx.Err()
}
```

---

## 17. Linter Configuration

### `.golangci.yml` (required in repo root)

```yaml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - errcheck          # no ignored errors
    - govet             # go vet checks
    - staticcheck       # advanced static analysis
    - gosimple          # simplify code
    - ineffassign       # assignment to variable never used
    - unused            # unused code
    - gochecknoinits    # no init() functions
    - gocritic          # opinionated checks
    - godot             # comments end with period
    - misspell          # spelling in comments
    - noctx             # HTTP request must have context
    - nilerr            # return nil, nil anti-pattern
    - exhaustive        # exhaustive enum switches
    - prealloc          # slice pre-allocation hints
    - revive            # replaces golint
    - wrapcheck         # errors from external packages must be wrapped

linters-settings:
  errcheck:
    check-blank: true       # flag `_ = f()` for non-whitelisted functions
    check-type-assertions: true

  govet:
    enable-all: true

  revive:
    rules:
      - name: exported           # every exported symbol needs a comment
      - name: var-naming
      - name: package-comments
      - name: error-return       # error last in return values
      - name: increment-decrement # use i++ not i += 1

  exhaustive:
    default-signifies-exhaustive: false  # switch must be exhaustive

issues:
  exclude-rules:
    # Allow missing comment on test helper functions
    - path: "_test\\.go"
      linters: [godot, revive]
    # Allow init in generated files
    - path: "mock_.*\\.go"
      linters: [gochecknoinits]
```

### Pre-commit hook

```bash
#!/bin/sh
# .githooks/pre-commit
set -e

echo "→ gofmt"
gofmt -l . | grep -v vendor | xargs -r false

echo "→ golangci-lint"
golangci-lint run --fast

echo "→ go test (unit only, no integration)"
go test -short ./...
```

Install: `git config core.hooksPath .githooks`

---

## Quick Reference Card

```
NAMING          — camelCase vars, PascalCase exports, interface as -er verbs
PACKAGES        — short lowercase, responsibility-based, no utils/
IMPORTS         — stdlib | third-party | internal (3 groups)
FUNCTIONS       — ≤ 30 lines, ≤ 4 params, guard clauses, explicit returns
ERRORS          — return sentinel, wrap with %w, log once at origin, never expose raw errors
CONTEXT         — first param always, never stored in struct
CONCURRENCY     — all goroutines cancel on ctx.Done(), no bare goroutines
DI              — constructor injection, interface at consumer, compile-time assertion
LAYERS          — domain → service → storage/api, no horizontal, no upward imports
TESTS           — table-driven, testify, testcontainers for storage, ≥ 75% on service
LOGGING         — slog only, structured fields, warn on 4xx, error on 5xx
SECURITY        — parameterised SQL, bcrypt cost 12, RS256 JWT, validate all input
ANTI-PATTERNS   — no globals, no init(), no ORM, no panic on expected errors, no _
```

---

*This document is authoritative. Any deviation requires an explicit ADR (Architecture Decision Record) in `docs/documents/ADR/`.*
