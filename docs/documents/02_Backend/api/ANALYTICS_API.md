# API Reference — Analytics Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `60_ANALYTICS/ANALYTICS.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).

> **Architecture note:** Analytics tables are **write-heavy, append-only, and partitioned by month**.  
> There are **no user-facing write endpoints** — all inserts happen internally in Go services as side-effects of reading comics/chapters.  
> Public endpoints are read-only stats surfaces for admin dashboards.  
> Scale path: migrate to ClickHouse / TimescaleDB when monthly row count exceeds ~100M.

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Page Views, Chapter Sessions, Dashboard stats. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Page Views](#2-page-views)
3. [Chapter Sessions](#3-chapter-sessions)
4. [Admin Dashboard Stats](#4-admin-dashboard-stats)
5. [Internal Write Paths](#5-internal-write-paths-go-services-only)
6. [Implementation Notes](#6-implementation-notes)

---

## Endpoint Summary

> All endpoints require `Authorization: Bearer <access_token>` with role `admin`.  
> There are **no public endpoints** in this domain.

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/analytics/pageviews` | Query raw page view events |
| `GET` | `/admin/analytics/pageviews/summary` | Aggregated view stats (total, unique, by date) |
| `GET` | `/admin/analytics/comics/:id/views` | View trend for a specific comic |
| `GET` | `/admin/analytics/chapters/:id/views` | View trend for a specific chapter |
| `GET` | `/admin/analytics/sessions` | Query chapter reading sessions |
| `GET` | `/admin/analytics/sessions/summary` | Aggregated session stats |
| `GET` | `/admin/analytics/chapters/:id/sessions` | Session stats for a specific chapter |
| `GET` | `/admin/analytics/dashboard` | Overall platform stats snapshot |
| `GET` | `/admin/analytics/top-comics` | Top comics by views / follows / rating |
| `GET` | `/admin/analytics/top-chapters` | Top chapters by views and completion rate |

---

## 1. Common Types

### `PageViewSummary`
```typescript
{
  entitytype: "comic" | "chapter"
  entityid: string
  totalviews: number
  uniqueusers: number       // distinct non-null userids
  anonymousviews: number    // userid = null
  bydate: Array<{ date: string; views: number }>
}
```

### `ChapterSessionSummary`
```typescript
{
  chapterid: string
  totalsessions: number
  completed: number          // sessions where lastpage >= chapter.pagecount
  completionrate: number     // completed / totalsessions (0.0–1.0)
  avgreadtimeseconds: number // avg(EXTRACT(EPOCH FROM finishedat - startedat)) where finishedat IS NOT NULL
  bydevice: {
    mobile: number
    desktop: number
    tablet: number
    unknown: number
  }
  bydate: Array<{ date: string; sessions: number; completionrate: number }>
}
```

### `DashboardSnapshot`
```typescript
{
  generatedat: string
  period: { from: string; to: string }
  views: {
    total: number
    comics: number
    chapters: number
    uniqueusers: number
    changepct: number       // % change vs previous period
  }
  users: {
    total: number
    newthisperiod: number
    activethisperiod: number   // users with >= 1 pageview
    changepct: number
  }
  content: {
    totalcomics: number
    totalchapters: number
    newchaptersthisperiod: number
  }
  sessions: {
    total: number
    avgcompletionrate: number
    avgreadtimeseconds: number
  }
}
```

---

## 2. Page Views

### GET /admin/analytics/pageviews

Query raw page view events. Useful for debugging or exporting data.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `entitytype` | string | — | `comic` \| `chapter` |
| `entityid` | string | — | Filter by specific entity UUID |
| `userid` | string | — | Filter by user. Use `anonymous` to show only `userid IS NULL`. |
| `from` | string | — | ISO 8601 start datetime |
| `to` | string | — | ISO 8601 end datetime. Default: `NOW()` |
| `page` | int | `1` | — |
| `limit` | int | `100` | Max `1000` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 10000001,
      "entitytype": "chapter",
      "entityid": "01952fa5-...",
      "userid": "01952fa3-...",
      "ipaddress": "203.0.113.42",
      "useragent": "Mozilla/5.0 ...",
      "referrer": "https://yomira.app/comics/solo-leveling",
      "createdat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 84231, "page": 1, "limit": 100, "pages": 843 }
}
```

> **Privacy:** `ipaddress` and `useragent` are **anonymized (set to NULL) after 90 days** by background job per privacy policy.

---

### GET /admin/analytics/pageviews/summary

Aggregated view stats — total, unique, anonymous, by date.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `entitytype` | string | — | `comic` \| `chapter` — if omitted, aggregates both |
| `from` | string | 30 days ago | ISO 8601 start datetime |
| `to` | string | `NOW()` | ISO 8601 end datetime |
| `granularity` | string | `day` | `hour` \| `day` \| `week` \| `month` |

**Response `200 OK`:**
```json
{
  "data": {
    "entitytype": null,
    "entityid": null,
    "totalviews": 4820341,
    "uniqueusers": 182043,
    "anonymousviews": 1204201,
    "bydate": [
      { "date": "2026-02-21", "views": 284021 },
      { "date": "2026-02-22", "views": 312400 }
    ]
  }
}
```

---

### GET /admin/analytics/comics/:id/views

View trend for a specific comic (includes both comic page views and its chapter views).

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — comic UUIDv7

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `from` | string | 30 days ago | ISO 8601 |
| `to` | string | `NOW()` | ISO 8601 |
| `granularity` | string | `day` | `hour` \| `day` \| `week` \| `month` |
| `includechapters` | bool | `true` | Include chapter view counts in totals |

**Response `200 OK`:**
```json
{
  "data": {
    "entitytype": "comic",
    "entityid": "01952fb0-...",
    "totalviews": 102843921,
    "uniqueusers": 4821043,
    "anonymousviews": 38204011,
    "bydate": [
      { "date": "2026-02-21", "views": 48200 },
      { "date": "2026-02-22", "views": 51300 }
    ]
  }
}
```

---

### GET /admin/analytics/chapters/:id/views

View trend for a specific chapter.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — chapter UUIDv7

**Query params:** `from`, `to`, `granularity` (same as comic views)

**Response `200 OK`:** Same shape as `GET /admin/analytics/comics/:id/views` with `entitytype: "chapter"`.

---

## 3. Chapter Sessions

### GET /admin/analytics/sessions

Query raw chapter reading sessions.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `chapterid` | string | — | Filter by chapter |
| `comicid` | string | — | Filter by all chapters of a comic |
| `userid` | string | — | Filter by user |
| `devicetype` | string | — | `mobile` \| `desktop` \| `tablet` \| `unknown` |
| `completed` | bool | — | `true` = sessions where `lastpage >= pagecount` |
| `from` | string | 7 days ago | ISO 8601 `startedat` range start |
| `to` | string | `NOW()` | ISO 8601 `startedat` range end |
| `page` | int | `1` | — |
| `limit` | int | `100` | Max `1000` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 20000001,
      "chapterid": "01952fa5-...",
      "userid": "01952fa3-...",
      "startedat": "2026-02-22T00:00:00Z",
      "finishedat": "2026-02-22T00:08:34Z",
      "lastpage": 17,
      "devicetype": "desktop"
    }
  ],
  "meta": { "total": 18204, "page": 1, "limit": 100, "pages": 183 }
}
```

---

### GET /admin/analytics/sessions/summary

Aggregated session stats across all chapters.

**Auth required:** Yes (role: `admin`)

**Query params:** `from`, `to`, `granularity`, `comicid` (optional)

**Response `200 OK`:**
```json
{
  "data": {
    "totalsessions": 284021,
    "completed": 201430,
    "completionrate": 0.709,
    "avgreadtimeseconds": 512,
    "bydevice": {
      "mobile": 148200,
      "desktop": 112040,
      "tablet": 18421,
      "unknown": 5360
    },
    "bydate": [
      { "date": "2026-02-21", "sessions": 14200, "completionrate": 0.71 },
      { "date": "2026-02-22", "sessions": 15300, "completionrate": 0.70 }
    ]
  }
}
```

---

### GET /admin/analytics/chapters/:id/sessions

Detailed session stats for a specific chapter.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — chapter UUIDv7

**Query params:** `from`, `to`, `granularity`

**Response `200 OK`:** Full `ChapterSessionSummary` object.

```json
{
  "data": {
    "chapterid": "01952fa5-...",
    "totalsessions": 84231,
    "completed": 61200,
    "completionrate": 0.727,
    "avgreadtimeseconds": 487,
    "bydevice": {
      "mobile": 44200, "desktop": 32400,
      "tablet": 6200, "unknown": 1431
    },
    "bydate": [
      { "date": "2026-02-21", "sessions": 4200, "completionrate": 0.73 }
    ]
  }
}
```

---

## 4. Admin Dashboard Stats

### GET /admin/analytics/dashboard

Overall platform stats snapshot for the admin dashboard.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `period` | string | `7d` | `24h` \| `7d` \| `30d` \| `90d` — comparison period |

**Response `200 OK`:** Full `DashboardSnapshot` object.

```json
{
  "data": {
    "generatedat": "2026-02-22T00:20:23Z",
    "period": { "from": "2026-02-15T00:00:00Z", "to": "2026-02-22T00:00:00Z" },
    "views": {
      "total": 12840321,
      "comics": 2014300,
      "chapters": 10826021,
      "uniqueusers": 482043,
      "changepct": 8.4
    },
    "users": {
      "total": 94821,
      "newthisperiod": 1204,
      "activethisperiod": 38204,
      "changepct": 12.1
    },
    "content": {
      "totalcomics": 52000,
      "totalchapters": 4820000,
      "newchaptersthisperiod": 4821
    },
    "sessions": {
      "total": 8240021,
      "avgcompletionrate": 0.71,
      "avgreadtimeseconds": 504
    }
  }
}
```

> Dashboard stats are **computed on-demand** (no materialized view in v1.0). For heavy traffic, add a Redis cache with a 5-minute TTL.

---

### GET /admin/analytics/top-comics

Top comics ranked by views, follows, or rating in a time period.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `by` | string | `views` | `views` \| `follows` \| `rating` \| `sessions` |
| `from` | string | 7 days ago | ISO 8601 |
| `to` | string | `NOW()` | ISO 8601 |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "rank": 1,
      "comic": {
        "id": "01952fb0-...", "title": "Solo Leveling",
        "slug": "solo-leveling", "coverurl": "..."
      },
      "views": 1204021,
      "sessions": 820400,
      "completionrate": 0.73,
      "newfollows": 8420,
      "ratingavg": 9.18
    }
  ]
}
```

---

### GET /admin/analytics/top-chapters

Top chapters by views and completion rate.

**Auth required:** Yes (role: `admin`)

**Query params:** `from`, `to`, `comicid` (optional filter), `limit` (max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "rank": 1,
      "chapter": {
        "id": "01952fa5-...", "comicid": "01952fb0-...",
        "comictitle": "Solo Leveling", "chapternumber": 180.0,
        "language": { "code": "en" }
      },
      "views": 284021,
      "sessions": 201430,
      "completionrate": 0.71,
      "avgreadtimeseconds": 512
    }
  ]
}
```

---

## 5. Internal Write Paths (Go Services Only)

> These are **not HTTP endpoints**. They are Go service calls that write to analytics tables as **async side-effects**. No external client ever calls these directly.

| Trigger | Table written | Details |
|---|---|---|
| `GET /comics/:id` | `analytics.pageview` | `entitytype = 'comic'`, `entityid = comicid`. Fire-and-forget goroutine. |
| `GET /chapters/:id` | `analytics.pageview` | `entitytype = 'chapter'`, `entityid = chapterid`. Fire-and-forget goroutine. |
| Reader opens chapter | `analytics.chaptersession` | `INSERT` with `startedat = NOW()`, `finishedat = NULL`. |
| Reader closes / last page | `analytics.chaptersession` | `UPDATE finishedat, lastpage` on the open session. |
| Reader page turn (periodic) | `analytics.chaptersession` | `UPDATE lastpage` every N pages (debounced, not every page). |

### Go implementation pattern

```go
// Fire-and-forget page view insert (non-blocking)
func (s *AnalyticsService) RecordPageView(ctx context.Context, ev PageViewEvent) {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        _ = s.repo.InsertPageView(ctx, ev) // errors silently discarded
    }()
}

// Also increment Redis counters for fast read
func (s *AnalyticsService) RecordPageView(ctx context.Context, ev PageViewEvent) {
    go func() {
        _ = s.repo.InsertPageView(ctx, ev)
        _ = s.redis.Incr(ctx, fmt.Sprintf("%s:%s:views", ev.EntityType, ev.EntityID))
    }()
}
```

> **Redis counters** (`comic:{id}:views`, `chapter:{id}:views`) are the fast path for `core.comic.viewcount` and `core.chapter.viewcount`. A background job periodically flushes Redis counters back to Postgres (`HGETALL` + `UPDATE`).

---

## 6. Implementation Notes

### Go validation (NOT SQL)

```go
// analytics.pageview
entitytype: "comic" | "chapter"
referrer:   truncated to 500 chars before insert

// analytics.chaptersession
devicetype: "mobile" | "desktop" | "tablet" | "unknown"
lastpage:   >= 1 (if not nil)
// Session INSERT on chapter open; UPDATE on close — 1 session per (userid, chapterid, startedat)
// For anonymous users, session tracking is best-effort (no userid join)
```

### Aggregation SQL patterns

**Completion rate:**
```sql
SELECT
    COUNT(*)                                           AS totalsessions,
    COUNT(*) FILTER (WHERE s.lastpage >= c.pagecount) AS completed,
    ROUND(
        COUNT(*) FILTER (WHERE s.lastpage >= c.pagecount)::NUMERIC
        / NULLIF(COUNT(*), 0), 3
    )                                                  AS completionrate
FROM analytics.chaptersession s
JOIN core.chapter c ON c.id = s.chapterid
WHERE s.chapterid = $1
  AND s.startedat BETWEEN $2 AND $3;
```

**Average read time (seconds):**
```sql
SELECT ROUND(
    AVG(EXTRACT(EPOCH FROM finishedat - startedat))
)::INT AS avgreadtimeseconds
FROM analytics.chaptersession
WHERE chapterid = $1
  AND finishedat IS NOT NULL
  AND startedat BETWEEN $2 AND $3;
```

**Views by day:**
```sql
SELECT
    DATE_TRUNC('day', createdat) AS date,
    COUNT(*)                     AS views
FROM analytics.pageview
WHERE entitytype = $1
  AND entityid   = $2
  AND createdat BETWEEN $3 AND $4
GROUP BY 1
ORDER BY 1;
```

### Privacy policy — IP anonymization

```go
// Background job — runs daily
// Anonymizes pageview rows older than 90 days
UPDATE analytics.pageview
SET ipaddress = NULL,
    useragent = NULL
WHERE createdat < NOW() - INTERVAL '90 days'
  AND (ipaddress IS NOT NULL OR useragent IS NOT NULL);
```

> Run on age-partitioned tables: apply only to old partitions to avoid full-table scans.

### Partition management

```sql
-- Create next month's partitions (run monthly):
CREATE TABLE analytics.pageview_2026_04
    PARTITION OF analytics.pageview
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE analytics.chaptersession_2026_04
    PARTITION OF analytics.chaptersession
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- Detach and archive old partitions (> 12 months):
ALTER TABLE analytics.pageview DETACH PARTITION analytics.pageview_2026_02;
-- pg_dump analytics.pageview_2026_02 → cold storage → DROP TABLE
```

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /admin/analytics/dashboard` | 5 min | `analytics:dashboard:{period}` |
| `GET /admin/analytics/top-comics` | 5 min | `analytics:top-comics:{by}:{from}:{to}` |
| `GET /admin/analytics/top-chapters` | 5 min | `analytics:top-chapters:{from}:{to}` |
| `GET /admin/analytics/pageviews/summary` | 2 min | `analytics:pv-summary:{from}:{to}:{gran}` |
| `GET /admin/analytics/comics/:id/views` | 2 min | `analytics:comic:{id}:views:{from}:{to}` |
| Raw pageview/session queries | No cache | — |
| `comic:{id}:views` (fast counter) | No TTL (Redis INCR) | Flushed to DB hourly |
