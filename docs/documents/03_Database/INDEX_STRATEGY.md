# Index Strategy — Yomira Database

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Database:** PostgreSQL 16

---

## Table of Contents

1. [Index Types Used](#1-index-types-used)
2. [users Schema](#2-users-schema)
3. [core Schema](#3-core-schema)
4. [library Schema](#4-library-schema)
5. [social Schema](#5-social-schema)
6. [crawler Schema](#6-crawler-schema)
7. [analytics Schema](#7-analytics-schema)
8. [system Schema](#8-system-schema)
9. [Index Health Queries](#9-index-health-queries)

---

## 1. Index Types Used

| Type | SQL | Best for |
|---|---|---|
| **B-tree** | `USING btree` (default) | Equality, range, ORDER BY, `<`, `>`, `BETWEEN` |
| **GIN** | `USING gin` | Full-text search (`tsvector`), JSONB, arrays, trigrams (`gin_trgm_ops`) |
| **BRIN** | `USING brin` | Append-only time-series tables (e.g. partitioned analytics) |
| **Hash** | `USING hash` | Pure equality lookups (rare — B-tree usually better) |

### Partial index

All soft-delete tables define partial indexes with `WHERE deletedat IS NULL`. This keeps the index small and fast — deleted rows are never included.

```sql
CREATE INDEX idx_core_comic_status ON core.comic (status)
    WHERE deletedat IS NULL;   -- ← partial: excludes deleted rows
```

### Composite index column order

Always put **most selective** (highest cardinality) or **equality** column first, range/sort column last:

```sql
-- For query: WHERE userid = $1 ORDER BY updatedat DESC
CREATE INDEX idx_library_entry_user_updated
    ON library.entry (userid, updatedat DESC);
--           ^equality first  ^sort last
```

---

## 2. users Schema

### `users.account`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `account_pkey` | `id` | B-tree | — | Primary key — auto-created |
| `account_email_uq` | `email` | B-tree | — | Unique constraint — login lookup |
| `account_username_uq` | `username` | B-tree | — | Unique constraint — profile URLs |
| `idx_users_account_email` | `email` | B-tree | `deletedat IS NULL` | Filtered login lookup (excludes deleted) |
| `idx_users_account_role` | `role` | B-tree | `deletedat IS NULL` | Admin user list by role |
| `idx_users_account_createdat` | `createdat` | B-tree | `deletedat IS NULL` | Chronological user list |
| `idx_users_account_uname_trgm` | `username` | GIN `gin_trgm_ops` | — | Fuzzy username search |

```sql
-- idx_users_account_uname_trgm (add via migration)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX CONCURRENTLY idx_users_account_uname_trgm
    ON users.account USING GIN (username gin_trgm_ops)
    WHERE deletedat IS NULL;
```

### `users.session`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `session_pkey` | `id` | B-tree | — | PK |
| `idx_users_session_tokenhash` | `tokenhash` | B-tree (unique) | — | Auth middleware lookup — must be O(1) |
| `idx_users_session_userid` | `userid` | B-tree | — | "List my sessions" + session revocation |
| `idx_users_session_expiresat` | `expiresat` | B-tree | — | Expired session cleanup job |

### `users.follow`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `follow_pkey` | `(followerid, followingid)` | B-tree | Composite PK — enforces uniqueness |
| `idx_users_follow_followingid` | `followingid` | B-tree | "List followers of user X" |

> The PK already covers `followerid` queries; `followingid` needs its own index.

---

## 3. core Schema

### `core.comic`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `comic_pkey` | `id` | B-tree | — | PK |
| `comic_slug_uq` | `slug` | B-tree | — | URL slug uniqueness |
| `idx_core_comic_status` | `status` | B-tree | `deletedat IS NULL` | Filter by ongoing/completed |
| `idx_core_comic_ratingbayesian` | `ratingbayesian DESC` | B-tree | `deletedat IS NULL` | Top-rated sort |
| `idx_core_comic_followcount` | `followcount DESC` | B-tree | `deletedat IS NULL` | Popular sort |
| `idx_core_comic_viewcount` | `viewcount DESC` | B-tree | `deletedat IS NULL` | Most viewed sort |
| `idx_core_comic_createdat` | `createdat DESC` | B-tree | `deletedat IS NULL` | Newest sort |
| `idx_core_comic_title_trgm` | `title` | GIN `gin_trgm_ops` | `deletedat IS NULL` | Fuzzy title search |

```sql
-- High-value composite: status + sort (covers the most common browse query)
CREATE INDEX idx_core_comic_status_rating
    ON core.comic (status, ratingbayesian DESC)
    WHERE deletedat IS NULL;
```

### `core.chapter`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `chapter_pkey` | `id` | B-tree | — | PK |
| `idx_core_chapter_comicid` | `(comicid, chapternumber DESC)` | B-tree | `deletedat IS NULL` | Chapter list for a comic, ordered |
| `idx_core_chapter_createdat` | `createdat DESC` | B-tree | `deletedat IS NULL` | Latest uploads feed |
| `idx_core_chapter_syncstate` | `syncstate` | B-tree | `syncstate != 'synced'` | Find pending/errored chapters |
| `idx_core_chapter_groupid` | `groupid` | B-tree | `deletedat IS NULL` | Chapters by scanlation group |

### `core.author` / `core.artist`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `idx_core_author_name_trgm` | `name` | GIN `gin_trgm_ops` | Fuzzy author name search |
| `idx_core_artist_name_trgm` | `name` | GIN `gin_trgm_ops` | Fuzzy artist name search |

### `core.comictag` (M:N join)

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `comictag_pkey` | `(comicid, tagid)` | B-tree | Composite PK |
| `idx_core_comictag_tagid` | `tagid` | B-tree | Filter comics by tag |

### `core.comicauthor` / `core.comicartist`

| Index | Columns | Rationale |
|---|---|---|
| `comicauthor_pkey` | `(comicid, authorid)` | PK — also covers lookup by comic |
| `idx_core_comicauthor_authorid` | `authorid` | Comics by author |

---

## 4. library Schema

### `library.entry`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `entry_pkey` | `id` | B-tree | — | PK |
| `entry_user_comic_uq` | `(userid, comicid)` | B-tree | — | Unique constraint — one entry per comic per user |
| `idx_library_entry_userid` | `(userid, updatedat DESC)` | B-tree | — | "My shelf" list, most recently updated first |
| `idx_library_entry_status` | `(userid, readingstatus)` | B-tree | — | Filter shelf by status |
| `idx_library_entry_hasnew` | `userid` | B-tree | `hasnew = TRUE` | "Comics with new chapters" badge |

### `library.chapterread`

| Index | Columns | Rationale |
|---|---|---|
| `chapterread_pkey` | `(userid, chapterid)` | Composite PK — enforces uniqueness, covers lookup |
| `idx_library_chapterread_chapter` | `chapterid` | Admin/analytics: who read a chapter |

### `library.readingprogress`

| Index | Columns | Rationale |
|---|---|---|
| `readingprogress_pkey` | `id` | PK |
| `readingprogress_user_comic_uq` | `(userid, comicid)` | Unique — one progress record per comic per user |

### `library.viewhistory`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `viewhistory_pkey` | `id` | B-tree | — | PK |
| `idx_library_viewhistory_user` | `(userid, viewedat DESC)` | B-tree | — | "My history" list by recency |
| `idx_library_viewhistory_user_comic` | `(userid, comicid)` | B-tree | — | Check if user has viewed a comic |

---

## 5. social Schema

### `social.comment`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `comment_pkey` | `id` | B-tree | — | PK |
| `idx_social_comment_comicid` | `(comicid, createdat DESC)` | B-tree | `isdeleted = FALSE` | Comic comment list |
| `idx_social_comment_chapterid` | `(chapterid, createdat DESC)` | B-tree | `isdeleted = FALSE` | Chapter comment list |
| `idx_social_comment_parentid` | `(parentid, createdat ASC)` | B-tree | `isdeleted = FALSE` | Replies to a comment |
| `idx_social_comment_userid` | `userid` | B-tree | — | "My comments" history |

### `social.notification`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `notification_pkey` | `id` | B-tree | — | PK |
| `idx_social_notification_userid` | `(userid, createdat DESC)` | B-tree | — | Notification inbox |
| `idx_social_notification_unread` | `userid` | B-tree | `isread = FALSE` | Unread count badge — O(1) |

```sql
-- The unread partial index is critical for the bell icon badge count
CREATE INDEX idx_social_notification_unread
    ON social.notification (userid)
    WHERE isread = FALSE;
-- Query: SELECT COUNT(*) FROM social.notification WHERE userid = $1 AND isread = FALSE
--        → Index Only Scan on the partial index (extremely fast)
```

### `social.feedevent`

| Index | Columns | Rationale |
|---|---|---|
| `feedevent_pkey` | `id` | PK |
| `idx_social_feedevent_user_created` | `(userid, createdat DESC)` | Cursor-paginated feed |

### `social.forumthread` / `social.forumpost`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `idx_social_forumthread_forum` | `(forumid, lastpostedat DESC)` | B-tree | Thread list in a forum |
| `idx_social_forumthread_search` | `searchvector` | GIN | Full-text search |
| `idx_social_forumpost_thread` | `(threadid, createdat ASC)` | B-tree | Posts in a thread (chronological) |
| `idx_social_forumpost_search` | `searchvector` | GIN | Full-text search |

---

## 6. crawler Schema

### `crawler.job`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `job_pkey` | `id` | B-tree | — | PK |
| `idx_crawler_job_status` | `(sourceid, status, scheduledat)` | B-tree | `status != 'completed'` | Find pending/running jobs |
| `idx_crawler_job_scheduledat` | `scheduledat` | B-tree | `status = 'pending'` | Scheduler: next jobs to run |

### `crawler.log`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `idx_crawler_log_jobid` | `(jobid, createdat DESC)` | B-tree | Log lines for a job |

> `crawler.log` is range-partitioned by month. BRIN indexes on `createdat` per partition are more efficient than B-tree for append-only data.

```sql
CREATE INDEX idx_crawler_log_createdat_brin
    ON crawler.log USING brin (createdat);
```

---

## 7. analytics Schema

`analytics.pageview` and `analytics.chaptersession` are **range-partitioned by month**. Indexes are created per-partition.

### `analytics.pageview`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `idx_analytics_pageview_date_brin` | `eventdate` | BRIN | Range scans on date — very efficient for append-only |
| `idx_analytics_pageview_comicid` | `(comicid, eventdate)` | B-tree | Per-comic view stats |
| `idx_analytics_pageview_chapterid` | `(chapterid, eventdate)` | B-tree | Per-chapter view stats |

```sql
-- BRIN index: 1 block = 128 pages. Much smaller than B-tree for time-series.
-- Efficient when data is physically ordered by insertion time (which it is for analytics).
CREATE INDEX idx_analytics_pageview_2026_02_date_brin
    ON analytics.pageview_2026_02 USING brin (eventdate);
```

### `analytics.chaptersession`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `idx_analytics_chaptersession_date_brin` | `startedat` | BRIN | Range scans |
| `idx_analytics_chaptersession_chapter` | `(chapterid, startedat)` | B-tree | Per-chapter session stats |

---

## 8. system Schema

### `system.auditlog`

| Index | Columns | Type | Rationale |
|---|---|---|---|
| `auditlog_pkey` | `id` | B-tree | PK |
| `idx_system_auditlog_actorid` | `(actorid, createdat DESC)` | B-tree | Audit history for a user |
| `idx_system_auditlog_action` | `(action, createdat DESC)` | B-tree | Filter by action type |

### `system.announcement`

| Index | Columns | Type | Partial | Rationale |
|---|---|---|---|---|
| `announcement_pkey` | `id` | B-tree | — | PK |
| `idx_system_announcement_active` | `publishedat DESC` | B-tree | `deletedat IS NULL AND isactive = TRUE` | Active announcements homepage |

---

## 9. Index Health Queries

### Find unused indexes

```sql
SELECT schemaname, tablename, indexname,
       idx_scan AS scans,
       pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
JOIN pg_index USING (indexrelid)
WHERE idx_scan < 10
  AND NOT indisprimary
  AND NOT indisunique
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Find missing indexes (sequential scans on large tables)

```sql
SELECT schemaname, relname AS tablename,
       seq_scan, idx_scan,
       pg_size_pretty(pg_total_relation_size(relid)) AS size
FROM pg_stat_user_tables
WHERE seq_scan > idx_scan
  AND pg_total_relation_size(relid) > 10 * 1024 * 1024  -- > 10MB
ORDER BY seq_scan DESC;
```

### Index bloat estimate

```sql
SELECT schemaname, tablename, indexname,
       pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
       round(100 * pg_relation_size(indexrelid) /
             nullif(pg_total_relation_size(relid), 0)) AS pct_of_table
FROM pg_stat_user_indexes
JOIN pg_index USING (indexrelid)
ORDER BY pg_relation_size(indexrelid) DESC
LIMIT 20;
```

### Rebuild a bloated index

```sql
-- Online reindex (no table lock — PostgreSQL 12+)
REINDEX INDEX CONCURRENTLY idx_core_comic_title_trgm;
```

### Check if all indexes fit in `shared_buffers`

```sql
SELECT pg_size_pretty(SUM(pg_relation_size(indexrelid))) AS total_index_size
FROM pg_stat_user_indexes;
-- Compare with: SHOW shared_buffers;
-- Ideally, hot indexes should fit in shared_buffers (default 128MB, tune to 25% of RAM)
```
