# ADR 001 — Why PostgreSQL

> **Status:** Accepted  
> **Date:** 2026-02-21  
> **Author:** tai.buivan.jp@gmail.com

---

## Context

Yomira needs a primary data store for user accounts, comic metadata, social interactions, reading progress, and analytics. The data is relational by nature (comics → chapters → pages, users → follows → comics, etc.) and requires ACID transactions for operations like rating upserts and session rotation.

We evaluated four options:

| Option | Type | Considered |
|---|---|---|
| **PostgreSQL 16** | RDBMS | ✅ Chosen |
| MongoDB | Document DB | Considered |
| MySQL / MariaDB | RDBMS | Considered |
| CockroachDB | Distributed SQL | Considered |

---

## Decision

**Use PostgreSQL 16** as the sole primary database.

---

## Reasons

### 1. Relational data fits a relational model

Comic metadata is inherently relational: a comic has multiple authors, artists, tags, titles in different languages, covers, chapters, pages. Trying to model this in a document DB leads to deeply nested documents, awkward updates, and eventual consistency problems.

### 2. Advanced features used directly in the app

Yomira uses PostgreSQL-specific features that are not available in MySQL or document databases:

| Feature | Used for |
|---|---|
| `pg_trgm` (GIN index) | Fuzzy comic/author/user search |
| `tsvector` / `websearch_to_tsquery` | Forum full-text search |
| `ON CONFLICT DO UPDATE` | Ratings, library entries, votes — atomic upsert |
| Range partitioning | `analytics.pageview` partitioned by month |
| GENERATED ALWAYS AS (stored) | `searchvector` computed columns |
| Window functions (`COUNT(*) OVER()`) | Single-query pagination with total count |
| `pg_stat_statements` | Query performance monitoring |
| Row-level security (planned v2) | Multi-tenant data isolation |

### 3. Schema-per-domain organization

PostgreSQL schemas (`users`, `core`, `library`, `social`, `crawler`, `analytics`, `system`) provide namespace isolation similar to microservice databases — without the operational overhead of running 7 separate databases. All schemas can be queried in a single transaction.

### 4. ACID transactions

Operations like "refresh token rotation" (revoke old session + create new session atomically) require ACID guarantees. Document databases either lack transactions or add complexity to achieve them.

### 5. golang-migrate compatibility

Go ecosystem has first-class support for PostgreSQL migrations via `golang-migrate`, `pgx`, and `sqlx`. The `pgx/v5` driver is the most performant and feature-complete Go PostgreSQL client.

### 6. Proven at scale

PostgreSQL handles billions of rows with proper indexing. Partitioned `analytics.pageview` can scale to hundreds of millions of rows per year. Read replicas can be added transparently for read-heavy workloads.

---

## Tradeoffs Accepted

| Tradeoff | Mitigation |
|---|---|
| Vertical scaling limit | Read replicas + connection pooling (PgBouncer) |
| Schema migrations require coordination | golang-migrate with zero-downtime patterns (see MIGRATION_GUIDE.md) |
| Full-text search weaker than Elasticsearch | pg_trgm acceptable for v1.0; Elasticsearch path documented in SEARCH_API.md |
| Single-region by default | Supabase / Neon for multi-region in v2.0 |

---

## Rejected Alternatives

### MongoDB
- Relational comic data is awkward in document model
- Joins are application-side (slow for complex queries)
- Transaction support added late, still complex to use correctly
- Weak typing: schema validation optional, errors at runtime

### MySQL / MariaDB
- No `pg_trgm` (trigram search requires workarounds)
- Weaker JSON support
- Less expressive window functions
- Go ecosystem prefers PostgreSQL (`pgx` vs `go-sql-driver/mysql`)

### CockroachDB
- Distributed SQL adds operational complexity unnecessary for v1.0
- Higher latency due to consensus protocol
- Some PostgreSQL features missing (window functions, certain index types)
- Cost: managed CockroachDB is more expensive than managed PostgreSQL
