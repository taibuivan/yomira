# Yomira — System Architecture

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-21  
> **Status:** Active

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Technology Stack](#3-technology-stack)
4. [Architecture Decision Records (ADRs)](#4-architecture-decision-records)
5. [Deployment Roadmap](#5-deployment-roadmap)
6. [Service Boundaries](#6-service-boundaries)
7. [Data Flow](#7-data-flow)
8. [Security Model](#8-security-model)

---

## 1. Project Overview

**Yomira** is a manga/manhwa/webtoon reading platform built for Vietnamese readers. Core features:

| Feature | Description |
|---|---|
| Comic catalog | Browse, search, filter manga/manhwa/webtoon |
| Multi-language chapters | Chapters in multiple languages from scanlation groups |
| Library | Personal reading shelf with progress tracking |
| Social | Ratings, comments, recommendations, forum, feed |
| Crawler | Automated content ingestion from external sources |
| Analytics | Reading behaviour and traffic analytics |

**Design philosophy:**
- **Monolith first** — single Go binary, single PostgreSQL instance. Scale incrementally.
- **Go at every layer** — server-side HTML rendering + REST API, no separate frontend build.
- **Validation in Go, not SQL** — DB enforces structure (PK/FK/NOT NULL). Go enforces business rules.
- **Named after schema, not technology** — packages are organized by domain responsibility.

---

## 2. High-Level Architecture

```
                 ┌─────────────────────────────────────────────┐
                 │              Yomira Go binary               │
                 │                                             │
  Browser ──────►  HTTP (net/http + chi router)                │
  Mobile App ───►  ├── Middleware (auth, CORS, logging)        │
  Crawler Bot ───►  ├── API handlers (/api/v1/...)             │
                 │  ├── HTML template handlers (SSR)           │
                 │  └── Static file server (/static/*)         │
                 │                                             │
                 │  Service Layer (domain logic)               │
                 │  ├── ComicService                           │
                 │  ├── UserService                            │
                 │  ├── LibraryService                         │
                 │  ├── SocialService                          │
                 │  └── CrawlerService                         │
                 │                                             │
                 │  Repository Layer (data access)             │
                 │  ├── pgx (PostgreSQL driver)                │
                 │  └── Redis client (cache + session)         │
                 └──────────────┬──────────────────────────────┘
                                │
              ┌─────────────────┼──────────────────┐
              │                 │                  │
        ┌─────▼──────┐   ┌──────▼─────┐   ┌──────▼──────┐
        │ PostgreSQL │   │   Redis    │   │ Object      │
        │  (primary) │   │  (cache /  │   │ Storage     │
        │            │   │  session / │   │ (S3/R2/MinIO│
        │ 7 schemas  │   │  counters) │   │ /local)     │
        └────────────┘   └────────────┘   └─────────────┘
```

---

## 3. Technology Stack

| Layer | Technology | Reason |
|---|---|---|
| **Language** | Go 1.22+ | Static typing, fast compilation, excellent concurrency, no runtime overhead |
| **HTTP Router** | `chi` | Lightweight, idiomatic middleware chaining, compatible with `net/http` |
| **Database** | PostgreSQL 15+ | ACID, schemas for namespacing, partitioning, tsvector FTS, extensions |
| **DB Driver** | `pgx/v5` | Fastest Go PostgreSQL driver, native protocol, supports `pgxpool` |
| **Migrations** | `golang-migrate` | SQL-based, `up`/`down` pairs, CLI integration |
| **Cache** | Redis 7 | Sessions, view-count buffering, feed fan-out (future) |
| **Object Storage** | Cloudflare R2 / MinIO | Images: cover, pages, media files |
| **Templating** | Go `html/template` | Built-in, XSS-safe, no build step for Phase 1 |
| **Auth** | JWT (RS256) | Stateless tokens; sessions stored in Redis/DB |
| **Password** | bcrypt (cost=12) | Industry standard; upgradeable via re-hash on login |
| **Logging** | `slog` (stdlib) | Structured JSON logs, zero-dep |
| **Config** | `envconfig` or `viper` | Env-var based config, 12-factor app compliance |
| **Crawler** | Go plugin system | Extensions loaded at runtime per source |

---

## 4. Architecture Decision Records

### ADR-01: Validation in Go, not SQL

**Status:** Accepted  
**Date:** 2026-02-21

**Context:** Enum-style constraints in SQL (`CHECK (role IN ('admin','member'))`) require `ALTER TABLE` to add new values, which locks the table and cannot be rolled back.

**Decision:** Remove all enum-style `CHECK` constraints. SQL enforces only:
- `PRIMARY KEY`
- `UNIQUE`
- `FOREIGN KEY`
- `NOT NULL`

Go service layer enforces:
- Allowed values (role, status, content rating, etc.)
- Range checks (score 1–10, preloadpages 1–10)
- Cross-field business rules

**Consequences:** Zero-downtime when adding new enum values. Single source of truth in Go domain constants.

---

### ADR-02: Single PostgreSQL Instance with Schema Namespacing

**Status:** Accepted  
**Date:** 2026-02-21

**Context:** Microservices from day one is over-engineering for a solo project. A monolith is easier to operate, debug, and evolve.

**Decision:** One PostgreSQL database with 7 logical schemas:

```
yomira_db
├── users.*      — accounts, sessions, OAuth, preferences, follow graph
├── core.*       — comics, chapters, pages, authors, tags, media files
├── library.*    — shelf, reading progress, custom lists, view history
├── social.*     — comments, ratings, notifications, feed, forum, reports
├── crawler.*    — sources, jobs, structured logs
├── analytics.*  — page views, reading sessions (write-heavy, partitioned)
└── system.*     — audit log, settings, announcements
```

**Consequences:** ACID cross-schema transactions. Normal SQL JOINs. Single `pg_dump`. Future migration path: extract any schema into its own PostgreSQL instance by changing the repository connection string.

---

### ADR-03: UUIDv7 as Primary ID for Public-Facing Tables

**Status:** Accepted  
**Date:** 2026-02-21

**Context:** Auto-increment integers expose row counts and are predictable. UUIDs v4 are random — bad for B-tree index fragmentation.

**Decision:**

| Table category | ID type | Reason |
|---|---|---|
| Public-facing, high-cardinality | UUIDv7 (TEXT) | Time-sortable → B-tree insert at end. Non-guessable → safe in URLs. |
| Lookup / reference data | SERIAL START 100000 | Compact, never public-facing |
| History / junction write-heavy | BIGSERIAL | Sequential scans by FK |

UUIDv7 is generated at the **application layer** (Go), not by PostgreSQL. This means the DB never needs `uuid-ossp` for these tables.

---

### ADR-04: Server-Side Rendering First, API Second

**Status:** Accepted  
**Date:** 2026-02-21

**Context:** SPA frameworks add build complexity, deployment steps, and hydration bugs. Our target users include low-bandwidth mobile users.

**Decision:** Phase 1–2 uses Go `html/template` for SSR. The REST API (`/api/v1/`) exists simultaneously and serves the same data. Mobile apps and future SPA can use the API directly.

**Consequences:** Faster initial page load. No hydration issues. Easier SEO. API can be developed independently of the UI.

---

### ADR-05: Pull-Based Feed (Phase 1), Push-Based (Phase 3+)

**Status:** Accepted  
**Date:** 2026-02-21

**Decision:**

```sql
-- Phase 1 (< 100k users): Pull on request
SELECT * FROM social.feedevent
WHERE entityid IN (
    SELECT comicid FROM library.entry WHERE userid = $me
    UNION
    SELECT groupid FROM core.scanlationgroupfollow WHERE userid = $me
)
ORDER BY createdat DESC LIMIT 20;

-- Phase 3 (> 1M users): Redis sorted set per user (push fan-out)
ZADD feed:{userid} {timestamp} {eventid}
```

---

## 5. Deployment Roadmap

```
Phase 1 — Bootstrap (< 10k users)
┌──────────────────────────────────────────────┐
│  1 VPS/server                                │
│  ├── Go binary (port 8080)                   │
│  ├── PostgreSQL 15 (local or managed)        │
│  ├── Redis 7 (local or managed)              │
│  └── Nginx (reverse proxy + TLS)             │
└──────────────────────────────────────────────┘

Phase 2 — Growth (10k–500k users)
┌──────────────────────────────────────────────┐
│  Go binary (auto-scale, 2–4 pods)            │
│  PostgreSQL primary + 1 read replica         │
│  Redis cluster (sessions + counters)         │
│  CDN for images (Cloudflare R2)              │
│  Managed PostgreSQL (e.g. Supabase/Neon)     │
└──────────────────────────────────────────────┘

Phase 3 — Scale (500k–5M users)
┌──────────────────────────────────────────────┐
│  analytics.* → ClickHouse (OLAP)             │
│  Image processing microservice               │
│  Search → Elasticsearch or pg_search         │
│  Feed → Redis fan-out (push model)           │
│  Background jobs → dedicated worker process  │
└──────────────────────────────────────────────┘

Phase 4 — Enterprise (5M+ users)
┌──────────────────────────────────────────────┐
│  Per-schema microservices                    │
│  Kafka event bus                             │
│  Dedicated crawler cluster                   │
│  Multi-region deployment                     │
└──────────────────────────────────────────────┘
```

---

## 6. Service Boundaries

```
internal/
├── domain/          ← Pure Go. No external dependencies. Structs + interfaces only.
│   ├── comic.go
│   ├── user.go
│   ├── library.go
│   ├── social.go
│   └── crawler.go
├── service/         ← Business logic. Depends on domain interfaces only.
│   ├── comic.go
│   ├── user.go
│   ├── library.go
│   └── ...
├── storage/         ← Implements domain interfaces using pgx. SQL lives here.
│   ├── postgres/
│   │   ├── comic_repo.go
│   │   ├── user_repo.go
│   │   └── ...
│   └── redis/
│       ├── session_store.go
│       └── counter.go
├── api/             ← HTTP handlers. Calls service layer. Knows nothing about storage.
│   ├── handler_comic.go
│   ├── handler_auth.go
│   └── ...
├── auth/            ← Token signing/validation. Bcrypt. Middleware.
├── crawler/         ← Extension loader + job scheduler.
├── config/          ← Config struct. Reads env vars once at startup.
└── worker/          ← Background jobs (counters flush, hasnew, partitions).
```

**Dependency rule:** `domain` ← `service` ← `api`. No layer reaches down more than one level.

---

## 7. Data Flow

### Request: User reads a chapter

```
Browser
  │  GET /comic/{id}/chapter/{chapterId}
  ▼
Nginx (TLS termination, rate limit)
  │
  ▼
Go HTTP server
  │
  ├── Middleware: auth (validates JWT → injects user into ctx)
  │
  ├── Handler: handler_chapter.go
  │     │  1. Call ChapterService.GetChapter(id)
  │     │  2. Call AnalyticsService.RecordView() — async goroutine
  │     │  3. Call LibraryService.UpdateProgress() — if authenticated
  │     │
  │     └──► ChapterService
  │               │  1. Check Redis cache (chapter:{id})
  │               │  2. Miss → query PostgreSQL core.chapter + core.page
  │               │  3. Fill Redis cache (TTL 5 min)
  │               └──► Return chapter + pages
  │
  ▼
Render HTML template (SSR) or return JSON (/api/v1/)
```

### Background: View count flush

```
Redis INCR comic:{id}:views
      │
      │  Every 60 seconds (worker goroutine)
      ▼
UPDATE core.comic SET viewcount = viewcount + $delta WHERE id = $id
```

---

## 8. Security Model

| Concern | Implementation |
|---|---|
| **Authentication** | JWT (RS256). Access token (15 min) + Refresh token (30 days). |
| **Session storage** | SHA-256 hash of refresh token stored in `users.session`. Raw token never stored. |
| **Password storage** | bcrypt, cost=12. Re-hashed on login if cost < current default. |
| **OAuth tokens** | Encrypted at rest (AES-256-GCM) in `users.oauthprovider.accesstoken`. |
| **Rate limiting** | Nginx + Go middleware: 60 req/min per IP for API. 10 req/min for auth endpoints. |
| **CSRF** | SameSite=Strict cookie. Double-submit token for HTML forms. |
| **XSS** | `html/template` auto-escapes. Content Security Policy header. |
| **SQL Injection** | `pgx` parameterized queries everywhere. No string concatenation in SQL. |
| **Input validation** | Go service layer validates all input before any DB call. |
| **Sensitive data** | `deletedat` soft-delete retains PII for legal. GDPR: export + delete endpoint planned. |
| **Audit trail** | All privileged actions written to `system.auditlog` (append-only, `ON DELETE SET NULL`). |
