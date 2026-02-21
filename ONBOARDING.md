# Onboarding Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> For new developers joining the Yomira project.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture in 5 Minutes](#2-architecture-in-5-minutes)
3. [Setup Checklist](#3-setup-checklist)
4. [Your First Week](#4-your-first-week)
5. [Key Concepts to Know](#5-key-concepts-to-know)
6. [Where to Find Things](#6-where-to-find-things)
7. [Common Gotchas](#7-common-gotchas)
8. [Getting Help](#8-getting-help)

---

## 1. Project Overview

**Yomira** is a manga/manhwa/manhua reading platform. Users can read comics, track their reading progress, follow scanlation groups, rate and comment on comics, and get recommendations.

**Tech stack:**
- **Backend:** Go 1.22 (REST API), chi router
- **Database:** PostgreSQL 16 (7 schemas)
- **Cache:** Redis 7
- **Object Storage:** Cloudflare R2 (images, pages)
- **Email:** Resend (transactional)
- **Auth:** JWT RS256 (access) + SHA-256 refresh tokens

---

## 2. Architecture in 5 Minutes

### Domain schemas (PostgreSQL)

```
10_USERS    → accounts, sessions, OAuth, follow graph, reading preferences
20_CORE     → languages, authors, artists, tags, scanlation groups, comics, chapters, pages
30_LIBRARY  → user shelves, custom lists, reading progress, chapter read history
40_SOCIAL   → ratings, comments, notifications, recommendations, feed, forum, reports
50_CRAWLER  → crawl sources, jobs, logs (admin/internal)
60_ANALYTICS → page views, chapter sessions (partitioned by month)
70_SYSTEM   → audit log, settings, announcements
```

### Request flow

```
Client → Cloudflare CDN/WAF
       → Load Balancer
       → Go API Server
           │ ← Middleware chain (RequestID → Logger → Recovery → CORS → RateLimit → Auth → RoleGuard)
           │ ← Handler (parse request, call service)
           │ ← Service layer (business logic, validation)
           │ ← Storage layer (SQL queries, Redis)
           └ → PostgreSQL / Redis / R2
```

### Layer rules (strict)

```
api handler → service interface → storage interface → DB/Redis
         ↑                  ↑ No skipping layers ↑
```

### Key files

```
cmd/server/main.go          ← entrypoint, router setup, middleware chain
src/server/api/             ← HTTP handlers
src/server/service/         ← business logic
src/server/storage/         ← SQL queries
src/server/middleware/      ← auth, rate limit, CORS, etc.
src/server/shared/          ← apperr, validate, respond, auth helpers
src/common/DML/             ← SQL schemas + migrations
docs/documents/             ← all documentation
```

---

## 3. Setup Checklist

### Day 1 — Get the code running

```bash
□ Clone repo
  git clone https://github.com/your-org/yomira.git && cd yomira

□ Install Go 1.22+
  go version   # verify

□ Install Docker Desktop
  docker --version   # verify

□ Install tools
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  go install github.com/air-verse/air@latest

□ Copy and configure .env
  cp .env.example .env
  # Set DATABASE_URL, REDIS_URL, JWT_PRIVATE_KEY_PATH, JWT_PUBLIC_KEY_PATH
  # Generate JWT keys if not provided:
  openssl genrsa -out secrets/jwt_private.pem 2048
  openssl rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem

□ Start infrastructure
  docker compose up -d postgres redis minio mailhog

□ Run migrations
  migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up

□ Seed initial data
  psql "${DATABASE_URL}" -f src/common/DML/DATA/INITIAL_DATA.sql

□ Start server
  air   # or: go run ./cmd/server

□ Verify it works
  curl http://localhost:8080/health
  curl http://localhost:8080/api/v1/languages
```

### Access local services

| Service | URL |
|---|---|
| Go API | http://localhost:8080 |
| API Health | http://localhost:8080/health |
| MinIO Console | http://localhost:9001 (admin/minioadmin) |
| Mailhog (email viewer) | http://localhost:8025 |

---

## 4. Your First Week

### Day 1–2: Read the docs

Read in this order:

1. 📄 `docs/documents/01_Architecture/ARCHITECTURE.md` — big picture
2. 📄 `docs/documents/02_Backend/BACKEND.md` — Go package structure
3. 📄 `docs/documents/03_Database/DATABASE.md` — schema overview
4. 📄 `docs/documents/02_Backend/api/API_CONVENTIONS.md` — API standards
5. 📄 `docs/documents/02_Backend/MIDDLEWARE.md` — request pipeline
6. 📄 `docs/documents/02_Backend/ERROR_HANDLING.md` — error patterns

### Day 3: Explore the codebase

```bash
# Read the main entrypoint
cat cmd/server/main.go

# Explore one complete domain (start with CORE — most representative)
ls src/server/api/comic/
ls src/server/service/comic/
ls src/server/storage/comic/

# Read one handler end-to-end
cat src/server/api/comic/handler.go
cat src/server/service/comic/service.go
cat src/server/storage/comic/repo.go
```

### Day 4–5: Make your first change

1. Pick a small task from the backlog (labeled `good-first-issue`)
2. Create a branch: `feat/your-task-name`
3. Make the change
4. Write or update tests
5. Run local CI checks:
   ```bash
   gofmt -w . && go vet ./... && golangci-lint run ./... && go test ./...
   ```
6. Open a PR (see [CONTRIBUTING.md](../../CONTRIBUTING.md))

---

## 5. Key Concepts to Know

### UUIDv7

All primary-key IDs for user-facing resources are **UUIDv7** — time-ordered, globally unique, generated at the application layer (not DB).

```go
import "github.com/your-org/yomira/src/server/shared/uuidv7"
id := uuidv7.New().String()   // "01952fa3-a1b2-7000-8000-abcdef123456"
```

### Soft deletes

Most tables have a `deletedat TIMESTAMPTZ` column. Deleting a resource sets `deletedat = NOW()` — the row is never actually removed. **Always filter `WHERE deletedat IS NULL`** in queries.

### Error handling pattern

```go
// Return sentinel errors from storage
return nil, ErrComicNotFound

// Handlers call RespondError — never write HTTP responses in service/storage
RespondError(w, r, err)
```

See `docs/documents/02_Backend/ERROR_HANDLING.md` for full details.

### Context-first

Every function that touches the DB or Redis takes `ctx context.Context` as the **first argument**. Always propagate it — never use `context.Background()` in handlers.

```go
func (s *ComicService) GetByID(ctx context.Context, id string) (*Comic, error)
```

### Migrations

Never edit an applied migration. Always create a new file. Use `CONCURRENTLY` for indexes. See `docs/documents/03_Database/MIGRATION_GUIDE.md`.

---

## 6. Where to Find Things

| I want to... | Look here |
|---|---|
| Understand an API endpoint | `docs/documents/02_Backend/api/{DOMAIN}_API.md` |
| Find the SQL schema | `src/common/DML/{NN}_{SCHEMA}/{SCHEMA}.sql` |
| Add a new endpoint | `src/server/api/` → `service/` → `storage/` |
| Write a migration | `src/common/DML/migrations/` |
| Understand error codes | `docs/documents/02_Backend/api/API_CONVENTIONS.md#5-error-format` |
| Check middleware order | `docs/documents/02_Backend/MIDDLEWARE.md#1-middleware-stack-overview` |
| Find a common SQL pattern | `docs/documents/03_Database/QUERY_PATTERNS.md` |
| Run CI checks locally | `CONTRIBUTING.md#10-ci-checks` |
| Debug a production incident | `docs/documents/05_Operations/RUNBOOK.md` |

---

## 7. Common Gotchas

### 1. Missing `WHERE deletedat IS NULL`

Easy to forget — adds bugs that look like ghost data:

```go
// ❌ Returns deleted comics
rows, err := db.QueryContext(ctx, `SELECT id FROM core.comic WHERE status = $1`, "ongoing")

// ✅ Correct
rows, err := db.QueryContext(ctx, `SELECT id FROM core.comic WHERE status = $1 AND deletedat IS NULL`, "ongoing")
```

### 2. Not closing `rows`

Leaks DB connections:

```go
rows, err := db.QueryContext(ctx, ...)
if err != nil { return err }
defer rows.Close()   // ← always defer immediately after checking err
```

### 3. Forgetting owner checks

IDOR vulnerability — always check that the caller owns the resource they're mutating:

```go
if comment.UserID != caller.ID && !auth.IsModOrAbove(caller.Role) {
    return apperr.Forbidden("...")
}
```

### 4. Using `context.Background()` in handlers

Breaks request cancellation and tracing:

```go
// ❌ Loses request context (timeout, cancel, request ID)
result, err := svc.DoThing(context.Background(), id)

// ✅ Correct
result, err := svc.DoThing(r.Context(), id)
```

### 5. Rate limit middleware uses Redis — if Redis is down, requests pass through

By design — don't rely on rate limiting as a security boundary, only as a UX protection.

### 6. `CREATE INDEX` locks the table without `CONCURRENTLY`

Always use `CONCURRENTLY` in migrations for large tables. See MIGRATION_GUIDE.md.

---

## 8. Getting Help

| Channel | Use for |
|---|---|
| `#dev-general` | General questions |
| `#dev-backend` | Go / API / DB questions |
| `#alerts` | Production incidents |
| GitHub Issues | Bug reports, feature requests |
| GitHub PR review | Code review and feedback |
| This repo's docs | Architecture decisions, patterns |

**When asking a question, include:**
- Request ID if it's an API issue — `X-Request-ID` response header
- Error message + stack trace
- What you tried
- Which file/function is involved
