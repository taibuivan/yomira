# Yomira — Database Design

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.1.0 — 2026-02-21  
> **PostgreSQL:** 15+  
> Changes are listed newest first.

---

## Table of Contents

1. [Schema Map](#1-schema-map)
2. [Changelog](#2-changelog)
3. [Architecture Decisions](#3-architecture-decisions)
4. [Index Strategy](#4-index-strategy)
5. [Migration Guide](#5-migration-guide)
6. [Common Query Patterns](#6-common-query-patterns)
7. [Denormalized Counters](#7-denormalized-counters)
8. [Partitioned Tables](#8-partitioned-tables)

---

## 1. Schema Map

**7 schemas, 48 tables** (as of v1.1.0)

| Schema | Tables | Purpose |
|---|---|---|
| `users` | `account` · `oauthprovider` · `session` · `follow` · `readingpreference` | Auth, profiles, social graph |
| `core` | `language` · `author` · `artist` · `taggroup` · `tag` · `scanlationgroup` · `scanlationgroupmember` · `scanlationgroupfollow` · `comic` · `comictitle` · `comicrelation` · `comicauthor` · `comicartist` · `comictag` · `comiccover` · `comicart` · `chapter` · `page` · `mediafile` | Content catalog |
| `library` | `entry` · `customlist` · `customlistitem` · `readingprogress` · `chapterread` · `viewhistory` | Personal shelf |
| `social` | `comicrating` · `comment` · `commentvote` · `notification` · `comicrecommendation` · `comicrecommendationvote` · `feedevent` · `forum` · `forumthread` · `forumpost` · `forumpostvote` · `report` | Community |
| `crawler` | `source` · `comicsource` · `job` · `log` | Content ingestion |
| `analytics` | `pageview` · `chaptersession` | Traffic & behaviour |
| `system` | `auditlog` · `setting` · `announcement` | Admin & ops |

---

## 2. Changelog

### [v1.1.0] — 2026-02-21

| Item | File | Description |
|---|---|---|
| `core.comictitle` | `20_CORE/CORE.sql` | Per-language title + description (multilingual). |
| `library.viewhistory` | `30_LIBRARY/LIBRARY.sql` | Browsing history. Powers "Recently Viewed". |
| `users.readingpreference.datasaver` | `10_USERS/USERS.sql` | Data Saver mode (compressed images). |
| `idx_core_chapter_group` | `20_CORE/CORE.sql` | Chapter list by scanlation group. |
| `idx_library_entry_updatedat` | `30_LIBRARY/LIBRARY.sql` | Shelf sort by recently updated. |

### [v1.0.0] — 2026-02-21

Initial release. Schema split into 8 DML files under numbered folders.  
Removed all `CHECK` enum constraints — validation delegated to Go service layer.  
Partitioned tables: `analytics.pageview`, `analytics.chaptersession`, `crawler.log`.

---

## 3. Architecture Decisions

### ADR-01: Validation in Go, not SQL

DB enforces: `PRIMARY KEY` · `UNIQUE` · `FOREIGN KEY` · `NOT NULL`  
Go enforces: allowed values, range checks, business rules.

```sql
-- Avoid: breaks zero-downtime deploys
role VARCHAR(20) CHECK (role IN ('admin','member','banned'))

-- Use: add new roles by shipping Go binary only
role VARCHAR(20) NOT NULL DEFAULT 'member'
-- Validated by: domain.UserRole.IsValid()
```

### ADR-02: ID Strategy

| Category | Type | Reason |
|---|---|---|
| Public-facing, high-cardinality | `TEXT` (UUIDv7, app-generated) | Time-sortable; non-guessable in URLs |
| Lookup / reference data | `INTEGER` (SERIAL START 100000) | Compact, never public-facing |
| Write-heavy history | `BIGINT` (BIGSERIAL) | Sequential scans by FK |

### ADR-03: Multilingual Titles

`core.comic.title` = canonical English title (or primary language).  
`core.comictitle` = per-language rows. Go fallback order: `user locale → "en" → comic.title`.

```
comictitle rows for "One Piece":
  (languageid=en, title="One Piece")
  (languageid=ja, title="ワンピース")
  (languageid=vi, title="Vua Hải Tặc")
```

### ADR-04: Partitioned Tables

`analytics.pageview`, `analytics.chaptersession`, `crawler.log` — range partitioned by month.  
Old partitions are dropped (not DELETEd) for zero-lock data pruning.  
No FK constraints on write-heavy partitioned tables (enforced in Go).

### ADR-05: Pull-Based Feed

```sql
-- Phase 1 (< 500k users): pull on request
SELECT * FROM social.feedevent
WHERE entityid IN (
    SELECT comicid FROM library.entry WHERE userid = $me
    UNION ALL
    SELECT groupid FROM core.scanlationgroupfollow WHERE userid = $me
)
ORDER BY createdat DESC LIMIT 20;

-- Phase 3 (> 1M users): Redis sorted set per user (push fan-out)
```

---

## 4. Index Strategy

| Table | Index columns | Type | Condition |
|---|---|---|---|
| `users.account` | `(email)`, `(role)`, `(createdat)` | B-tree | `WHERE deletedat IS NULL` |
| `users.session` | `(tokenhash)` | Unique | — |
| `users.follow` | `(followingid)` | B-tree | — |
| `core.author/artist` | `(name)` | GIN/trigram | — |
| `core.comic` | `(searchvector)` | GIN | — |
| `core.comic` | `(title)` | GIN/trigram | — |
| `core.comic` | `(status)`, `(contentrating)`, `(demographic)`, `(originlanguage)` | B-tree | `WHERE deletedat IS NULL` |
| `core.comic` | `(viewcount DESC)`, `(followcount DESC)`, `(ratingavg DESC)`, `(year)` | B-tree | `WHERE deletedat IS NULL` |
| `core.comictitle` | `(comicid)`, `(title)` trigram | B-tree / GIN | — |
| `core.chapter` | `(comicid, chapternumber)`, `(comicid, languageid)` | B-tree | `WHERE deletedat IS NULL` |
| `core.chapter` | `(scanlationgroupid, comicid, chapternumber)` | B-tree | `WHERE deletedat IS NULL` |
| `core.chapter` | `(publishedat DESC)`, `(viewcount DESC)`, `(syncstate)` | B-tree | `WHERE deletedat IS NULL` |
| `library.entry` | `(userid, readingstatus)`, `(userid, updatedat DESC)` | B-tree | — |
| `library.entry` | `(userid)` | B-tree | `WHERE hasnew = TRUE` |
| `library.viewhistory` | `(userid, viewedat DESC)` | B-tree | — |
| `library.chapterread` | `(userid)`, `(chapterid)` | B-tree | — |
| `social.comment` | `(comicid, createdat DESC)`, `(chapterid, createdat DESC)` | B-tree | `WHERE NOT isdeleted` |
| `social.notification` | `(userid, isread, createdat DESC)`, `(userid)` | B-tree | `WHERE isread = FALSE` |
| `social.comicrecommendation` | `(fromcomicid, upvotes DESC)`, `(tocomicid)` | B-tree | — |
| `social.feedevent` | `(entitytype, entityid)`, `(createdat)` | B-tree | — |
| `social.forumthread` | `(forumid, lastpostedat DESC)` | B-tree | `WHERE NOT isdeleted` |
| `social.report` | `(status)` | B-tree | `WHERE status IN ('open','reviewing')` |
| `crawler.job` | `(status)`, `(sourceid)`, `(scheduledat)` | B-tree | — |
| `analytics.pageview` | `(entitytype, entityid, createdat)` | B-tree | — |

---

## 5. Migration Guide

### File execution order

```
00_SETUP/SETUP.sql
10_USERS/USERS.sql
20_CORE/CORE.sql
30_LIBRARY/LIBRARY.sql
40_SOCIAL/SOCIAL.sql
50_CRAWLER/CRAWLER.sql
60_ANALYTICS/ANALYTICS.sql
70_SYSTEM/SYSTEM.sql
DATA/INITIAL_DATA.sql
```

### Naming convention (golang-migrate)

```
migrations/
├── 000001_setup.up.sql
├── 000001_setup.down.sql
├── 000002_users.up.sql
├── 000002_users.down.sql
...
├── 000010_add_comictitle.up.sql       ← v1.1.0 additive
└── 000010_add_comictitle.down.sql
```

**Rules:**
- Never edit a migration that has been applied — create a new one.
- `down.sql` must be the perfect inverse of `up.sql`.
- Comment header: `-- v1.1.0 | 2026-02-21 | Add core.comictitle table`

```bash
# Apply all pending
migrate -database "$DATABASE_URL" -path ./migrations up

# Rollback last migration
migrate -database "$DATABASE_URL" -path ./migrations down 1

# Check current version
migrate -database "$DATABASE_URL" -path ./migrations version
```

---

## 6. Common Query Patterns

### Latest chapter updates (homepage)

```sql
SELECT c.id, c.title, c.coverurl,
       ch.id AS chapter_id, ch.chapternumber, ch.publishedat,
       l.name AS language, sg.name AS group_name
FROM core.chapter ch
JOIN core.comic           c  ON c.id  = ch.comicid
JOIN core.language        l  ON l.id  = ch.languageid
LEFT JOIN core.scanlationgroup sg ON sg.id = ch.scanlationgroupid
WHERE ch.deletedat IS NULL
  AND c.deletedat  IS NULL
  AND l.code = $1
ORDER BY ch.publishedat DESC
LIMIT 24 OFFSET $2;
```

### Full-text search with filters

```sql
SELECT c.id, c.title, c.coverurl, c.status,
       c.ratingbayesian, c.followcount,
       ts_rank(c.searchvector, q) AS rank
FROM core.comic c,
     websearch_to_tsquery('simple', unaccent($1)) AS q
WHERE c.searchvector @@ q
  AND c.deletedat IS NULL
  AND ($2::varchar IS NULL OR c.status = $2)
  AND ($3::varchar IS NULL OR c.contentrating = $3)
ORDER BY rank DESC, c.followcount DESC
LIMIT 24 OFFSET $4;
```

### Activity Feed

```sql
SELECT fe.id, fe.eventtype, fe.entitytype, fe.entityid,
       fe.actorid, fe.payload, fe.createdat
FROM social.feedevent fe
WHERE fe.entityid IN (
    SELECT comicid FROM library.entry WHERE userid = $1
    UNION ALL
    SELECT groupid FROM core.scanlationgroupfollow WHERE userid = $1
)
ORDER BY fe.createdat DESC
LIMIT 20 OFFSET $2;
```

### Continue Reading

```sql
SELECT e.comicid, e.lastreadchapterid,
       ch.chapternumber, ch.title AS chapter_title
FROM library.entry e
JOIN core.chapter ch ON ch.id = e.lastreadchapterid
WHERE e.userid = $1
  AND e.readingstatus = 'reading'
  AND e.lastreadchapterid IS NOT NULL
ORDER BY e.lastreadat DESC
LIMIT 10;
```

---

## 7. Denormalized Counters

| Counter | Table | Update strategy |
|---|---|---|
| `viewcount` | `core.comic` | Redis `INCR` → worker flushes every 60s |
| `followcount` | `core.comic` | Direct `UPDATE ± 1` on follow/unfollow |
| `chaptercount` | `core.comic` | Direct `UPDATE ± 1` on chapter publish/delete |
| `ratingavg` | `core.comic` | Running avg: `(avg * n + new) / (n + 1)` |
| `ratingbayesian` | `core.comic` | `(C*m + Σscores) / (C+n)` — background job every 15 min |
| `ratingcount` | `core.comic` | Direct `UPDATE ± 1` on rating upsert |
| `threadcount` | `social.forum` | Direct `UPDATE ± 1` on thread create/delete |
| `postcount` | `social.forum` | Direct `UPDATE ± 1` on post create/delete |
| `replycount` | `social.forumthread` | Direct `UPDATE ± 1` on post create/delete |

---

## 8. Partitioned Tables

Tables are range-partitioned by month:

```sql
-- Create next month's partition (run on 1st of each month)
CREATE TABLE analytics.pageview_2026_04
    PARTITION OF analytics.pageview
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- Drop old partition (zero-lock, instant)
DROP TABLE analytics.pageview_2025_10;
```

| Table | Key | Retention |
|---|---|---|
| `analytics.pageview` | `createdat` | 6 months raw; then aggregate |
| `analytics.chaptersession` | `startedat` | 6 months |
| `crawler.log` | `createdat` | 3 months |

> Partitioned tables have **no FK constraints** on `userid`/`chapterid` — enforced in Go to avoid partition pruning overhead on every insert.
