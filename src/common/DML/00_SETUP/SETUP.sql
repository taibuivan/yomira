-- =============================================================================
-- YOMIRA — 00_SETUP
-- Extensions · Schemas · Shared Helper Functions
-- @author  tai.buivan.jp@gmail.com
-- Version : 3.0.0  |  Updated : 2026-02-21
--
-- Run this file FIRST before any schema file.
-- Idempotent: safe to re-run (IF NOT EXISTS / OR REPLACE).
-- =============================================================================

-- ---------------------------------------------------------------------------
-- EXTENSIONS
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";   -- uuid_generate_v4() fallback
CREATE EXTENSION IF NOT EXISTS "pg_trgm";     -- trigram GIN for partial-match search
CREATE EXTENSION IF NOT EXISTS "btree_gin";   -- GIN on scalar columns
CREATE EXTENSION IF NOT EXISTS "unaccent";    -- accent-insensitive search (Vietnamese/Japanese)

-- ---------------------------------------------------------------------------
-- SCHEMAS  (logical namespaces — still one PostgreSQL instance)
--
--   users.*     accounts, sessions, oauth, preferences, follows
--   core.*      comics, chapters, pages, authors, tags, media, scanlation
--   library.*   user shelf, reading progress, custom lists
--   social.*    comments, ratings, notifications, feed, forum, reports
--   crawler.*   source sites, crawl jobs, logs
--   analytics.* page views, reading sessions  (write-heavy, isolatable)
--   system.*    audit log, settings, announcements
-- ---------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS users;
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS library;
CREATE SCHEMA IF NOT EXISTS social;
CREATE SCHEMA IF NOT EXISTS crawler;
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE SCHEMA IF NOT EXISTS system;

-- ---------------------------------------------------------------------------
-- HELPER: set_updatedat  — auto-stamp updatedat on every UPDATE
--
-- Attached to every mutable table via attach_updatedat_trigger().
-- Exceptions (append-only): system.auditlog, crawler.log — no updatedat column.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updatedat()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updatedat = NOW();
    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION set_updatedat() IS
    'Trigger function: sets updatedat = NOW() before every UPDATE row.
     Attached to all mutable tables via attach_updatedat_trigger().';

-- ---------------------------------------------------------------------------
-- HELPER: attach_updatedat_trigger  — convenience macro
--
-- Usage: SELECT attach_updatedat_trigger(''users'', ''account'');
-- Creates: TRIGGER trg_account_updatedat BEFORE UPDATE ON users.account
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION attach_updatedat_trigger(schema_name TEXT, table_name TEXT)
RETURNS VOID LANGUAGE plpgsql AS $$
BEGIN
    EXECUTE format(
        'CREATE TRIGGER trg_%2$s_updatedat
         BEFORE UPDATE ON %1$s.%2$s
         FOR EACH ROW EXECUTE FUNCTION set_updatedat()',
        schema_name, table_name
    );
END;
$$;

COMMENT ON FUNCTION attach_updatedat_trigger(TEXT, TEXT) IS
    'Attaches the set_updatedat trigger to a table.
     Call once per table during schema creation.
     Args: schema_name, table_name';

-- =============================================================================
-- VALIDATION POLICY
-- =============================================================================
--
-- All enum-style business rules (allowed values for VARCHAR columns like
-- status, role, format, etc.) are enforced EXCLUSIVELY at the Go service layer.
--
-- The database enforces only structural integrity:
--   PRIMARY KEY · UNIQUE · FOREIGN KEY · NOT NULL
--
-- Why: PostgreSQL CHECK constraints on enum values are non-transactional when
-- altered (ALTER TYPE / ADD VALUE cannot run inside a transaction), making
-- zero-downtime deployments impossible. Go domain constants are the single
-- source of truth for allowed values.
--
-- Example (Go):
--   type ComicStatus string
--   const (
--       ComicStatusOngoing   ComicStatus = "ongoing"
--       ComicStatusCompleted ComicStatus = "completed"
--       ...
--   )
--   func (s ComicStatus) IsValid() bool { ... }
--
-- =============================================================================

-- =============================================================================
-- END OF SETUP
-- =============================================================================
