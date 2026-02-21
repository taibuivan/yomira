# SQL Query Patterns — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Database:** PostgreSQL 16  
> **Applies to:** `src/storage/`

---

## Table of Contents

1. [Soft Delete Filter](#1-soft-delete-filter)
2. [Upsert (ON CONFLICT)](#2-upsert-on-conflict)
3. [Offset Pagination](#3-offset-pagination)
4. [Cursor Pagination](#4-cursor-pagination)
5. [Partial Update (PATCH)](#5-partial-update-patch)
6. [Bayesian Rating Formula](#6-bayesian-rating-formula)
7. [Trigram Search](#7-trigram-search)
8. [Full-Text Search](#8-full-text-search)
9. [Delta Counters](#9-delta-counters)
10. [Batch Insert](#10-batch-insert)
11. [Table Partitioning Queries](#11-table-partitioning-queries)
12. [EXPLAIN ANALYZE Tips](#12-explain-analyze-tips)

---

## 1. Soft Delete Filter

All tables with `deletedat` column use a **partial index** on `deletedat IS NULL`. Always include this condition in queries.

```sql
-- Always filter soft-deleted rows
SELECT id, title, status
FROM core.comic
WHERE id = $1
  AND deletedat IS NULL;

-- ✅ Uses partial index: idx_core_comic_status (WHERE deletedat IS NULL)
SELECT id, title FROM core.comic
WHERE status = 'ongoing'
  AND deletedat IS NULL
ORDER BY updatedat DESC
LIMIT 24;

-- ❌ Missing filter — returns deleted rows
SELECT id, title FROM core.comic WHERE status = 'ongoing';
```

```go
// In Go — define a package-level constant for reuse
const softDeleteFilter = "AND deletedat IS NULL"

// Or use a WHERE builder
func (r *ComicRepo) baseFilter() string {
    return "WHERE deletedat IS NULL"
}
```

---

## 2. Upsert (ON CONFLICT)

Used for rating updates, library entries, votes, progress tracking.

### Insert or update (full upsert)

```sql
-- library.entry upsert
INSERT INTO library.entry (userid, comicid, readingstatus, score, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (userid, comicid) DO UPDATE
    SET readingstatus = EXCLUDED.readingstatus,
        score         = EXCLUDED.score,
        notes         = EXCLUDED.notes,
        updatedat     = NOW();
```

### Insert or update only specific columns

```sql
-- social.comicrating upsert — only update score, preserve createdat
INSERT INTO social.comicrating (userid, comicid, score)
VALUES ($1, $2, $3)
ON CONFLICT (userid, comicid) DO UPDATE
    SET score     = EXCLUDED.score,
        updatedat = NOW();
```

### Insert or ignore (idempotent action)

```sql
-- library.chapterread — mark chapter as read (idempotent)
INSERT INTO library.chapterread (userid, chapterid)
VALUES ($1, $2)
ON CONFLICT (userid, chapterid) DO NOTHING;
```

### Conditional update on conflict

```sql
-- Only update if existing value is lower:
INSERT INTO social.comicrating (userid, comicid, score)
VALUES ($1, $2, $3)
ON CONFLICT (userid, comicid) DO UPDATE
    SET score = GREATEST(EXCLUDED.score, social.comicrating.score);
```

---

## 3. Offset Pagination

Standard pagination for list endpoints.

```sql
-- Pattern:
SELECT ..., COUNT(*) OVER() AS total_count
FROM core.comic
WHERE deletedat IS NULL
  AND ($sort_conditions)
ORDER BY updatedat DESC
LIMIT $limit OFFSET ($page - 1) * $limit;
```

```go
// Go implementation
func (r *ComicRepo) List(ctx context.Context, p PageParams) ([]Comic, int, error) {
    offset := (p.Page - 1) * p.Limit

    rows, err := r.db.QueryContext(ctx, `
        SELECT c.id, c.title, c.status,
               COUNT(*) OVER() AS total_count
        FROM core.comic c
        WHERE c.deletedat IS NULL
        ORDER BY c.updatedat DESC
        LIMIT $1 OFFSET $2
    `, p.Limit, offset)

    var comics []Comic
    var total int
    for rows.Next() {
        var c Comic
        rows.Scan(&c.ID, &c.Title, &c.Status, &total)
        comics = append(comics, c)
    }
    return comics, total, rows.Err()
}

// Compute meta
func PageMeta(total, page, limit int) map[string]int {
    pages := (total + limit - 1) / limit
    return map[string]int{
        "total": total, "page": page,
        "limit": limit, "pages": pages,
    }
}
```

> **Performance note:** `COUNT(*) OVER()` adds minimal overhead for small result sets. For tables > 100M rows, consider a separate `SELECT COUNT(*)` with a Redis-cached estimate.

---

## 4. Cursor Pagination

Used for time-series feeds and logs where rows can be inserted between page loads.

```sql
-- GET /me/feed?before=2026-02-22T00:00:00Z&limit=20

SELECT id, eventtype, payload, createdat
FROM social.feedevent
WHERE userid = $1
  AND ($before IS NULL OR createdat < $2)   -- cursor condition
ORDER BY createdat DESC
LIMIT $3;
```

```go
type FeedPage struct {
    Items      []FeedEvent
    NextBefore *time.Time   // nil = last page
}

func (r *FeedRepo) List(ctx context.Context, userID string, before *time.Time, limit int) (FeedPage, error) {
    var beforeParam interface{} = nil
    if before != nil {
        beforeParam = *before
    }

    rows, _ := r.db.QueryContext(ctx, `
        SELECT id, eventtype, payload, createdat
        FROM social.feedevent
        WHERE userid = $1
          AND ($2::timestamptz IS NULL OR createdat < $2)
        ORDER BY createdat DESC
        LIMIT $3
    `, userID, beforeParam, limit+1)   // fetch limit+1 to detect next page

    var items []FeedEvent
    for rows.Next() {
        var e FeedEvent
        rows.Scan(&e.ID, &e.EventType, &e.Payload, &e.CreatedAt)
        items = append(items, e)
    }

    var nextBefore *time.Time
    if len(items) > limit {
        t := items[limit-1].CreatedAt   // last item of actual page
        nextBefore = &t
        items = items[:limit]           // trim the +1
    }
    return FeedPage{Items: items, NextBefore: nextBefore}, rows.Err()
}
```

---

## 5. Partial Update (PATCH)

PATCH requests update only provided fields. Build the SET clause dynamically.

```go
// Go — safe dynamic SET builder
type ComicUpdateParams struct {
    Title       *string
    Description *string
    Status      *string
    Year        *int
}

func buildUpdateQuery(params ComicUpdateParams) (string, []interface{}) {
    setClauses := []string{"updatedat = NOW()"}
    args := []interface{}{}
    i := 1

    if params.Title != nil {
        setClauses = append(setClauses, fmt.Sprintf("title = $%d", i))
        args = append(args, *params.Title)
        i++
    }
    if params.Description != nil {
        setClauses = append(setClauses, fmt.Sprintf("description = $%d", i))
        args = append(args, *params.Description)
        i++
    }
    if params.Status != nil {
        setClauses = append(setClauses, fmt.Sprintf("status = $%d", i))
        args = append(args, *params.Status)
        i++
    }

    // Final $N is the ID
    args = append(args, comicID)
    query := fmt.Sprintf(
        `UPDATE core.comic SET %s WHERE id = $%d AND deletedat IS NULL`,
        strings.Join(setClauses, ", "), i,
    )
    return query, args
}
```

> If no fields are provided, return early without touching the DB.

---

## 6. Bayesian Rating Formula

Prevents low-vote-count comics from dominating the rating leaderboard.

```sql
-- Bayesian weighted average:
-- ((C * m) + (R * v)) / (C + v)
-- Where:
--   v = comic's vote count
--   R = comic's arithmetic average
--   C = global minimum votes threshold (e.g. 100)
--   m = global average rating across all comics

-- Recalculate for one comic (called after each rating change):
WITH global AS (
    SELECT AVG(score) AS global_avg,
           100 AS min_votes        -- C (constant)
    FROM social.comicrating
)
UPDATE core.comic
SET
    ratingavg       = (
        SELECT AVG(score)
        FROM social.comicrating
        WHERE comicid = $1
    ),
    ratingbayesian  = (
        SELECT ((g.min_votes * g.global_avg) + (COUNT(cr.score) * AVG(cr.score)))
               / (g.min_votes + COUNT(cr.score))
        FROM social.comicrating cr, global g
        WHERE cr.comicid = $1
    ),
    ratingcount     = (
        SELECT COUNT(*) FROM social.comicrating WHERE comicid = $1
    )
WHERE id = $1;
```

> **Background job** recalculates all comics nightly to stay in sync with any deleted ratings. Single-comic recalc happens synchronously on each `PUT /comics/:id/rating`.

---

## 7. Trigram Search

pg_trgm similarity search for comics, authors, users.

```sql
-- Ensure extension and index exist (see MIGRATION_GUIDE.md)
-- CREATE EXTENSION pg_trgm;
-- CREATE INDEX idx_core_comic_title_trgm ON core.comic USING GIN (title gin_trgm_ops);

-- Basic similarity search
SELECT id, title, similarity(title, $1) AS score
FROM core.comic
WHERE title % $1          -- uses GIN index (fast)
  AND deletedat IS NULL
ORDER BY score DESC, followcount DESC
LIMIT 24;

-- Combined prefix + trigram (for quick search autocomplete)
SELECT id, title, coverurl, followcount
FROM core.comic
WHERE (
    title ILIKE $1 || '%'                   -- prefix match (FAST, B-tree)
    OR similarity(title, $1) > 0.3          -- fuzzy match (GIN)
)
AND deletedat IS NULL
ORDER BY
    CASE
        WHEN LOWER(title) = LOWER($1) THEN 0          -- exact
        WHEN LOWER(title) LIKE LOWER($1) || '%' THEN 1 -- prefix
        ELSE 2                                          -- fuzzy
    END,
    followcount DESC
LIMIT 8;

-- Set threshold per query (default 0.3)
SELECT set_limit(0.25);     -- lower = more results, less precise
```

---

## 8. Full-Text Search

Used for forum thread and post search (better than trigram for natural language).

```sql
-- GIN index on generated tsvector:
-- ALTER TABLE social.forumthread
--   ADD COLUMN searchvector tsvector
--   GENERATED ALWAYS AS (to_tsvector('english', title)) STORED;
-- CREATE INDEX idx_forumthread_search ON social.forumthread USING GIN (searchvector);

-- Basic full-text search
SELECT id, title,
    ts_rank(searchvector, query)        AS rank,
    ts_headline('english', title, query) AS snippet
FROM social.forumthread,
     websearch_to_tsquery('english', $1) AS query
WHERE searchvector @@ query
  AND isdeleted = FALSE
ORDER BY rank DESC
LIMIT 20;

-- Combined title + body search (forumpost)
SELECT p.id, p.body,
    ts_rank(p.searchvector, query) AS rank
FROM social.forumpost p,
     websearch_to_tsquery('english', $1) AS query
WHERE p.searchvector @@ query
  AND p.isdeleted = FALSE
ORDER BY rank DESC
LIMIT 20;
```

---

## 9. Delta Counters

Efficiently increment/decrement denormalized counters using `+1`/`-1` deltas instead of `COUNT(*)` recalculation.

```sql
-- Increment comment upvotes when user votes
UPDATE social.comment
SET upvotes = upvotes + 1
WHERE id = $1;

-- Decrement when vote removed
UPDATE social.comment
SET upvotes = GREATEST(upvotes - 1, 0)   -- prevent negative
WHERE id = $1;

-- Swap vote: downvote removed, upvote added (one query)
UPDATE social.comment
SET
    upvotes   = upvotes   + CASE WHEN $2 = 1  THEN 1 ELSE 0 END,  -- add if new vote is up
    downvotes = downvotes + CASE WHEN $2 = -1 THEN 1 ELSE 0 END,  -- add if new vote is down
    upvotes   = upvotes   - CASE WHEN $3 = 1  THEN 1 ELSE 0 END,  -- sub if old vote was up
    downvotes = downvotes - CASE WHEN $3 = -1 THEN 1 ELSE 0 END   -- sub if old vote was down
WHERE id = $1;
```

> Delta counters can drift under concurrent load. The nightly background job (`library.recalc` / `social.recalc`) recalculates exact counts and corrects any drift.

---

## 10. Batch Insert

Use `pgx` batch or unnest for bulk inserts.

```go
// Bulk insert comic pages using unnest (fastest approach)
func (r *PageRepo) BulkInsert(ctx context.Context, pages []Page) error {
    ids       := make([]string, len(pages))
    numbers   := make([]int, len(pages))
    urls      := make([]string, len(pages))
    chapterID := pages[0].ChapterID

    for i, p := range pages {
        ids[i]     = uuidv7.New().String()
        numbers[i] = p.PageNumber
        urls[i]    = p.ImageURL
    }

    _, err := r.db.ExecContext(ctx, `
        INSERT INTO core.page (id, chapterid, pagenumber, imageurl)
        SELECT unnest($1::text[]), $2, unnest($3::int[]), unnest($4::text[])
        ON CONFLICT (chapterid, pagenumber) DO UPDATE
            SET imageurl = EXCLUDED.imageurl
    `, ids, chapterID, numbers, urls)
    return err
}
```

---

## 11. Table Partitioning Queries

`analytics.pageview` and `analytics.chaptersession` are range-partitioned by month.

```sql
-- Partition pruning works when filtering by eventdate/endedat
-- PostgreSQL automatically prunes non-matching partitions

-- ✅ Partition pruning — only scans 2026-02 partition
SELECT COUNT(*) FROM analytics.pageview
WHERE eventdate >= '2026-02-01'
  AND eventdate <  '2026-03-01';

-- ❌ No partition pruning — scans ALL partitions
SELECT COUNT(*) FROM analytics.pageview
WHERE EXTRACT(MONTH FROM eventdate) = 2;   -- function prevents pruning

-- Dashboard: last 30 days (crosses month boundary — scans 2 partitions)
SELECT DATE_TRUNC('day', eventdate) AS day, COUNT(*) AS views
FROM analytics.pageview
WHERE eventdate >= NOW() - INTERVAL '30 days'
GROUP BY 1
ORDER BY 1;
```

### Creating next month's partition (scheduled, see BATCH_API.md)

```sql
-- Run on 1st of each month to create next month's partition
CREATE TABLE IF NOT EXISTS analytics.pageview_2026_03
    PARTITION OF analytics.pageview
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

---

## 12. EXPLAIN ANALYZE Tips

### Check if an index is being used

```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT id, title FROM core.comic
WHERE title % 'solo leveling'
  AND deletedat IS NULL
ORDER BY similarity(title, 'solo leveling') DESC
LIMIT 10;
```

**Look for:**
- `Index Scan` or `Bitmap Index Scan` → index used ✅
- `Seq Scan` on a large table → missing index ❌
- `Buffers: hit=N` → data in cache ✅
- `Buffers: read=N` → disk reads (consider caching) ⚠️
- `rows=X (actual) vs rows=Y (estimated)` — large discrepancy → run `ANALYZE table_name`

### Identify slow queries

```sql
-- Find the 10 slowest queries in pg_stat_statements
SELECT query, calls, total_exec_time, mean_exec_time, rows
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_stat%'
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Force/forbid index usage (testing only)

```sql
-- Disable sequential scan to test if index helps
SET enable_seqscan = OFF;
EXPLAIN ANALYZE SELECT ...;
SET enable_seqscan = ON;  -- always reset!
```

### Update statistics after bulk load

```sql
-- After large data imports, update planner statistics
ANALYZE core.comic;
ANALYZE analytics.pageview;
```
