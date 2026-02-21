# Database Migration Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Tool:** [golang-migrate](https://github.com/golang-migrate/migrate)  
> **Applies to:** `src/common/DML/migrations/`

---

## Table of Contents

1. [Overview](#1-overview)
2. [Directory Structure](#2-directory-structure)
3. [Naming Convention](#3-naming-convention)
4. [Writing a Migration](#4-writing-a-migration)
5. [Running Migrations](#5-running-migrations)
6. [Zero-Downtime Patterns](#6-zero-downtime-patterns)
7. [Rollback Strategy](#7-rollback-strategy)
8. [Testing Migrations](#8-testing-migrations)
9. [Common Pitfalls](#9-common-pitfalls)

---

## 1. Overview

Yomira uses **golang-migrate** with sequential versioned SQL files. Every schema change — column additions, index creation, data backfills — must go through a migration file.

**Rules:**
- One logical change per migration file
- Every `up` migration must have a corresponding `down` migration
- Migrations run sequentially — never edit an already-applied migration
- All migrations must be tested locally before merge

---

## 2. Directory Structure

```
src/common/DML/
├── migrations/
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   ├── 000002_add_readingpref_datasaver.up.sql
│   ├── 000002_add_readingpref_datasaver.down.sql
│   ├── 000003_add_comic_ratingbayesian.up.sql
│   ├── 000003_add_comic_ratingbayesian.down.sql
│   └── ...
├── 10_USERS/USERS.sql     ← source of truth (current state)
├── 20_CORE/CORE.sql
└── CHANGELOG.md
```

---

## 3. Naming Convention

```
{version}_{description}.{direction}.sql

Version:     6-digit zero-padded integer (000001, 000002 ...)
Description: lowercase, underscores only, max 60 chars, describe the change
Direction:   up | down
```

**Examples:**

```
000001_initial_schema.up.sql
000002_add_readingpref_datasaver.up.sql
000003_add_comic_trigram_index.up.sql
000004_create_analytics_partition_2026q1.up.sql
000005_backfill_comic_slug.up.sql
000006_add_social_feedevent_search.up.sql
```

**Bad names:**
```
000003_fix.up.sql              ❌ too vague
000004_update comic.up.sql     ❌ spaces not allowed
000005_AddComicIndex.up.sql    ❌ PascalCase not allowed
```

---

## 4. Writing a Migration

### Adding a nullable column (safe)

```sql
-- 000010_add_comic_coverurlhd.up.sql
-- Add HD cover URL column to core.comic.
-- Nullable initially; populated by background job.

ALTER TABLE core.comic
    ADD COLUMN IF NOT EXISTS coverurlhd TEXT;

COMMENT ON COLUMN core.comic.coverurlhd
    IS 'Full-resolution HD cover (840×1200). NULL until back-filled.';
```

```sql
-- 000010_add_comic_coverurlhd.down.sql
ALTER TABLE core.comic DROP COLUMN IF EXISTS coverurlhd;
```

---

### Adding an index (CONCURRENTLY — no table lock)

```sql
-- 000011_add_trigram_index_comic_title.up.sql
-- Add pg_trgm GIN index for comic title search.
-- CONCURRENTLY avoids locking the table during index build.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_core_comic_title_trgm
    ON core.comic USING GIN (title gin_trgm_ops)
    WHERE deletedat IS NULL;
```

> **Important:** `CREATE INDEX CONCURRENTLY` cannot run inside a transaction. golang-migrate wraps statements in transactions by default. Use `-- migrate-no-transaction` at the top of the file.

```sql
-- 000011_add_trigram_index_comic_title.up.sql
-- migrate-no-transaction

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_core_comic_title_trgm
    ON core.comic USING GIN (title gin_trgm_ops)
    WHERE deletedat IS NULL;
```

```sql
-- 000011_add_trigram_index_comic_title.down.sql
-- migrate-no-transaction

DROP INDEX CONCURRENTLY IF EXISTS idx_core_comic_title_trgm;
```

---

### Adding a NOT NULL column (multi-step — zero-downtime)

See [§6 Zero-Downtime Patterns](#6-zero-downtime-patterns).

---

### Creating a new table

```sql
-- 000012_create_emailpref_table.up.sql
-- Separate email preferences from readingpreference JSONB into typed table.

CREATE TABLE users.emailpref (
    userid          TEXT            NOT NULL,
    enabled         BOOLEAN         NOT NULL DEFAULT TRUE,
    new_chapter     BOOLEAN         NOT NULL DEFAULT TRUE,
    comment_reply   BOOLEAN         NOT NULL DEFAULT TRUE,
    follow          BOOLEAN         NOT NULL DEFAULT FALSE,
    announcement    BOOLEAN         NOT NULL DEFAULT TRUE,
    digest_weekly   BOOLEAN         NOT NULL DEFAULT FALSE,
    updatedat       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT emailpref_pkey       PRIMARY KEY (userid),
    CONSTRAINT emailpref_account_fk FOREIGN KEY (userid)
        REFERENCES users.account (id) ON DELETE CASCADE
);
SELECT attach_updatedat_trigger('users', 'emailpref');
```

```sql
-- 000012_create_emailpref_table.down.sql
DROP TABLE IF EXISTS users.emailpref;
```

---

### Data backfill migration

```sql
-- 000013_backfill_comic_slug.up.sql
-- Back-fill slug column using title for comics where slug IS NULL.
-- Run in batches to avoid long-running transaction.

DO $$
DECLARE
    batch_size INT := 1000;
    offset_val INT := 0;
    rows_updated INT;
BEGIN
    LOOP
        UPDATE core.comic
        SET slug = LOWER(REGEXP_REPLACE(title, '[^a-zA-Z0-9]+', '-', 'g'))
        WHERE id IN (
            SELECT id FROM core.comic
            WHERE slug IS NULL
            LIMIT batch_size
        );

        GET DIAGNOSTICS rows_updated = ROW_COUNT;
        EXIT WHEN rows_updated = 0;

        PERFORM pg_sleep(0.1);   -- brief pause between batches
    END LOOP;
END $$;
```

---

## 5. Running Migrations

### Install golang-migrate CLI

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Environment variable

```bash
# .env
DATABASE_URL=postgres://user:pass@localhost:5432/yomira?sslmode=disable
```

### Commands

```bash
# Apply all pending migrations
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up

# Apply exactly N migrations
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 2

# Roll back one migration
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1

# Roll back all migrations (DESTRUCTIVE — dev only)
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down

# Check current migration version
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations version

# Force a version (fix dirty migrations)
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations force 5
```

### Programmatic (Go) — applied at server startup

```go
// cmd/server/main.go
func runMigrations(db *sql.DB) error {
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        return fmt.Errorf("migration driver: %w", err)
    }
    m, err := migrate.NewWithDatabaseInstance(
        "file://src/common/DML/migrations",
        "postgres", driver,
    )
    if err != nil {
        return fmt.Errorf("migration setup: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("migration failed: %w", err)
    }
    return nil
}
```

The `schema_migrations` table (created by golang-migrate) tracks the current version and a `dirty` flag. If a migration fails partway, `dirty = true` — fix the issue, then run `migrate force {version}`.

---

## 6. Zero-Downtime Patterns

**Problem:** Some schema changes lock the table and block reads/writes. In production with live traffic, this is unacceptable.

---

### Pattern 1: Add a nullable column → backfill → add NOT NULL constraint

```sql
-- Step 1: Add nullable column (instant, no lock)
-- 000020_add_user_timezone_step1.up.sql
ALTER TABLE users.readingpreference
    ADD COLUMN IF NOT EXISTS timezone TEXT;  -- nullable first
```

```sql
-- Step 2: Backfill data (batched, no table lock)
-- 000021_add_user_timezone_step2.up.sql
UPDATE users.readingpreference
SET timezone = 'UTC'
WHERE timezone IS NULL;
```

```sql
-- Step 3: Add NOT NULL + DEFAULT (after backfill is complete)
-- 000022_add_user_timezone_step3.up.sql
ALTER TABLE users.readingpreference
    ALTER COLUMN timezone SET NOT NULL,
    ALTER COLUMN timezone SET DEFAULT 'UTC';
```

---

### Pattern 2: Rename a column (without downtime)

Never use `ALTER TABLE RENAME COLUMN` directly — it breaks running code.

```sql
-- Step 1: Add new column
ALTER TABLE core.comic ADD COLUMN titleslug TEXT;

-- Step 2: Write to BOTH old (slug) and new (titleslug) from application code

-- Step 3: Backfill new column from old
UPDATE core.comic SET titleslug = slug WHERE titleslug IS NULL;

-- Step 4: Add NOT NULL
ALTER TABLE core.comic ALTER COLUMN titleslug SET NOT NULL;

-- Step 5: Drop old column (after all app instances deployed to use new name)
ALTER TABLE core.comic DROP COLUMN slug;
```

---

### Pattern 3: Create index without locking

Always use `CONCURRENTLY` and `-- migrate-no-transaction`:

```sql
-- migrate-no-transaction
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_core_comic_titleslug
    ON core.comic (titleslug)
    WHERE deletedat IS NULL;
```

---

### Pattern 4: Drop a table safely

```sql
-- Step 1: Remove all application references to the table (deploy)
-- Step 2: Only then drop the table:
DROP TABLE IF EXISTS legacy_table_name;
```

---

## 7. Rollback Strategy

### Automatic rollback in CI

```yaml
# .github/workflows/migrate.yml
- name: Run migrations
  run: migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up
  
- name: Run tests (if fail, migrations will be rolled back)
  run: go test ./...
  
- name: Rollback on test failure
  if: failure()
  run: migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1
```

### Manual rollback

```bash
# Roll back the last migration
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1

# Verify
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations version
```

### When a migration fails and leaves `dirty = true`

```bash
# 1. Check which version is dirty
migrate -database "${DATABASE_URL}" ... version

# 2. Fix the SQL error in the migration file (or the data issue)

# 3. Force the version back to the last clean version
migrate -database "${DATABASE_URL}" ... force {N-1}

# 4. Re-apply
migrate -database "${DATABASE_URL}" ... up 1
```

---

## 8. Testing Migrations

### Local test procedure

```bash
# 1. Apply migration
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1

# 2. Verify the change
psql "${DATABASE_URL}" -c "\d core.comic"   # inspect table structure

# 3. Test rollback
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1

# 4. Verify rollback
psql "${DATABASE_URL}" -c "\d core.comic"

# 5. Re-apply for development
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
```

### Integration test with testcontainers

```go
// tests/integration/migration_test.go
func TestMigrations(t *testing.T) {
    ctx := context.Background()

    // Spin up fresh PostgreSQL
    pg, _ := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image: "postgres:16",
            Env:   map[string]string{"POSTGRES_PASSWORD": "test"},
        },
        Started: true,
    })
    dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")

    // Run all migrations up
    m, _ := migrate.New("file://../../migrations", dsn)
    require.NoError(t, m.Up())

    // Assert current version
    v, dirty, _ := m.Version()
    assert.False(t, dirty)
    assert.Greater(t, v, uint(0))

    // Roll back all migrations
    require.NoError(t, m.Down())

    // Assert clean rollback
    _, _, err := m.Version()
    assert.ErrorIs(t, err, migrate.ErrNilVersion)
}
```

---

## 9. Common Pitfalls

| Pitfall | Problem | Fix |
|---|---|---|
| `ADD COLUMN ... NOT NULL` (no default) | Locks table for rewrite on non-empty table | Add nullable first, backfill, then add constraint |
| `CREATE INDEX` without `CONCURRENTLY` | Table-level lock for duration of build | Always use `CONCURRENTLY` + `-- migrate-no-transaction` |
| Editing an applied migration | Breaks migration version hash checks | Always create a new migration; never edit applied ones |
| `DROP COLUMN` on live code | Application code referencing old column will break | Remove code references first, deploy, then drop column |
| No `down` migration | Cannot roll back in incident | Always write the `down` file even if trivial |
| Long-running data migration in transaction | Holds locks, blocks table | Use batch loop with `pg_sleep`; run outside transaction |
| Using `TRUNCATE` in migration | Irreversible data loss | Use `DELETE` with conditions; never `TRUNCATE` |
| Not testing `down` migration | Rollback fails in production | Always test both `up` and `down` locally |
