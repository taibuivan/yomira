# API Reference — Batch Jobs & Background Workers

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`

> This document covers:
> 1. **HTTP trigger endpoints** — Admin-only endpoints to manually kick off, inspect, or cancel batch operations.
> 2. **Scheduled Go workers** — Background jobs run by the Go scheduler (cron-style). These have **no HTTP endpoint** — they are documented here for operational reference.
>
> All HTTP endpoints require `admin` role. Scheduled workers run in-process within the Go backend.

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. All background workers and admin trigger endpoints. |

---

## Table of Contents

1. [Batch Job Common Types](#1-batch-job-common-types)
2. [Library Batch Jobs](#2-library-batch-jobs)
3. [Content & Media Batch Jobs](#3-content--media-batch-jobs)
4. [Analytics Batch Jobs](#4-analytics-batch-jobs)
5. [System Maintenance Batch Jobs](#5-system-maintenance-batch-jobs)
6. [Admin — Batch Control Endpoints](#6-admin--batch-control-endpoints)
7. [Scheduled Job Registry](#7-scheduled-job-registry)
8. [Implementation Notes](#8-implementation-notes)

---

## Endpoint Summary

> All HTTP endpoints require `admin` role.

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/batch/jobs` | List all batch job runs |
| `GET` | `/admin/batch/jobs/:jobKey` | Get history for a specific job type |
| `POST` | `/admin/batch/jobs/:jobKey/run` | Manually trigger a batch job |
| `POST` | `/admin/batch/jobs/:jobKey/cancel` | Cancel a running batch job |
| `GET` | `/admin/batch/schedule` | View current job schedule |
| `PATCH` | `/admin/batch/schedule/:jobKey` | Enable/disable or change cron expression |
| `POST` | `/admin/batch/library/hasnew` | Trigger `hasnew` recalculation for all users |
| `POST` | `/admin/batch/library/hasnew/:comicId` | Trigger `hasnew` recalculation for one comic |
| `POST` | `/admin/batch/analytics/flush-counters` | Flush Redis view counters to Postgres |
| `POST` | `/admin/batch/analytics/anonymize` | Trigger IP/UA anonymization on old pageviews |
| `POST` | `/admin/batch/analytics/partition` | Create next month's analytics partitions |
| `POST` | `/admin/batch/crawler/partitions` | Create next month's crawler log partitions |
| `POST` | `/admin/batch/storage/cleanup` | Trigger orphaned media file cleanup |
| `POST` | `/admin/batch/sessions/cleanup` | Expire old user sessions |
| `POST` | `/admin/batch/history/cap` | Cap view history to 500 entries/user |
| `POST` | `/admin/batch/comics/ratings` | Recalculate all comic Bayesian ratings |
| `POST` | `/admin/batch/comics/counts` | Recalculate denormalized chapter/follow counts |
| `POST` | `/admin/batch/announcements/expire` | Hide expired announcements |

---

## 1. Batch Job Common Types

### `BatchJobRun`
```typescript
{
  id: string                     // UUIDv7 — unique per run
  jobkey: string                 // e.g. "library.hasnew" | "analytics.flush_counters"
  status: "queued" | "running" | "done" | "failed" | "cancelled"
  triggeredby: "scheduler" | "admin"
  triggeredbyuserid: string | null
  startedat: string | null
  finishedat: string | null
  durationms: number | null
  rowsaffected: number | null
  lasterror: string | null
  meta: object | null            // job-specific output (counts, stats, etc.)
  createdat: string
}
```

### `ScheduleEntry`
```typescript
{
  jobkey: string
  description: string
  cron: string                   // cron expression, e.g. "0 * * * *"
  isenabled: boolean
  lastrun: BatchJobRun | null
  nextrunat: string | null
}
```

---

## 2. Library Batch Jobs

### `library.hasnew` — Recalculate "Has New Chapters" Flag

**Trigger:** Scheduled + manual  
**Tables:** `library.entry`, `library.chapterread`, `core.chapter`  
**Frequency:** Every 15 minutes (or on new chapter upload)

**What it does:**
```sql
-- For every entry where we need to recalculate:
UPDATE library.entry SET hasnew = (
    EXISTS (
        SELECT 1 FROM core.chapter ch
        WHERE ch.comicid = library.entry.comicid
          AND ch.deletedat IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM library.chapterread cr
              WHERE cr.userid = library.entry.userid
                AND cr.chapterid = ch.id
          )
    )
)
WHERE userid = $userid AND comicid = $comicid;
```

**Triggered automatically:**
- When a new chapter is uploaded to a comic (`POST /chapters`)
- When a user marks a chapter as read (`POST /chapters/:id/read`, `POST /me/comics/:id/read-all`)
- Scheduled sweep every 15 min to catch any drift

---

### POST /admin/batch/library/hasnew

Manually trigger a full `hasnew` recalculation sweep for all users and comics.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "userid": null, "comicid": null }
```

| Field | Type | Notes |
|---|---|---|
| `userid` | string \| null | If provided, only recalculates for this user. |
| `comicid` | string \| null | If provided, only recalculates entries for this comic. |

**Response `202 Accepted`:**
```json
{
  "data": {
    "id": "01953100-...",
    "jobkey": "library.hasnew",
    "status": "queued",
    "triggeredby": "admin",
    "createdat": "2026-02-22T00:26:32Z"
  }
}
```

---

### POST /admin/batch/library/hasnew/:comicId

Trigger `hasnew` recalculation for all followers of a single comic (e.g. after a chapter upload issue).

**Auth required:** Yes (role: `admin`)

**Response `202 Accepted`:** `BatchJobRun` object.

---

### `library.viewhistory_cap` — Cap View History per User

**Trigger:** Scheduled  
**Tables:** `library.viewhistory`  
**Frequency:** Daily at 02:00 UTC

**What it does:**
```sql
-- Delete oldest entries beyond 500 per user
DELETE FROM library.viewhistory
WHERE id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (PARTITION BY userid ORDER BY viewedat DESC) AS rn
        FROM library.viewhistory
    ) ranked
    WHERE ranked.rn > 500
);
```

---

### POST /admin/batch/history/cap

Manually trigger view history cap.

**Auth required:** Yes (role: `admin`)

**Request body:** `{ "userid": null }` — if `null`, processes all users.

**Response `202 Accepted`:** `BatchJobRun` object.

---

## 3. Content & Media Batch Jobs

### `storage.orphan_cleanup` — Delete Orphaned Media Files

**Trigger:** Scheduled  
**Tables:** `core.mediafile`  
**Frequency:** Weekly at 03:00 UTC Sunday

**What it does:** Finds `core.mediafile` rows where the file is no longer referenced by any `core.comiccover`, `core.comicart`, `core.page`, or `users.account.avatarurl`. Deletes from object storage (R2/S3/MinIO) then removes the DB row.

```go
// Go pseudocode
files := repo.FindOrphanedMediaFiles(ctx, olderThan: 24*time.Hour)
for _, f := range files {
    _ = storage.Delete(ctx, f.StorageKey)
    _ = repo.DeleteMediaFile(ctx, f.ID)
}
```

---

### POST /admin/batch/storage/cleanup

Manually trigger orphaned media file cleanup.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "olderthan": "24h" }
```

| Field | Type | Default | Notes |
|---|---|---|---|
| `olderthan` | string | `"24h"` | Only clean files older than this duration. Go parses as `time.Duration`. |

**Response `202 Accepted`:** `BatchJobRun` object.

---

### `comics.ratings_recalc` — Recalculate Bayesian Ratings

**Trigger:** Scheduled  
**Tables:** `social.comicrating`, `core.comic`  
**Frequency:** Hourly

**What it does:**
```sql
UPDATE core.comic c SET
    ratingavg      = stats.avg_score,
    ratingbayesian = (
        -- Wilson / Bayesian: (C * m + sum_score) / (C + count)
        -- where C = global avg, m = minimum votes threshold (e.g. 100)
        ($C * $m + stats.sum_score) / ($m + stats.count)
    ),
    ratingcount    = stats.count
FROM (
    SELECT comicid,
           AVG(score)::NUMERIC(4,2) AS avg_score,
           SUM(score)               AS sum_score,
           COUNT(*)                 AS count
    FROM social.comicrating
    GROUP BY comicid
) stats
WHERE c.id = stats.comicid;
```

**POST /admin/batch/comics/ratings** — Manual trigger.

**Response `202 Accepted`:** `BatchJobRun` with `meta: { comicsUpdated: 52000 }`.

---

### `comics.counts_recalc` — Recalculate Denormalized Counts

**Trigger:** Scheduled  
**Tables:** `core.comic`, `core.chapter`, `library.entry`, `core.scanlationgroupfollow`  
**Frequency:** Every 30 minutes

**What it does:**
```sql
-- chapter count
UPDATE core.comic c SET chaptercount = (
    SELECT COUNT(*) FROM core.chapter ch
    WHERE ch.comicid = c.id AND ch.deletedat IS NULL
);

-- follow count (library.entry = comic follow)
UPDATE core.comic c SET followcount = (
    SELECT COUNT(*) FROM library.entry e
    WHERE e.comicid = c.id
);

-- scanlation group follow count
UPDATE core.scanlationgroup sg SET followcount = (
    SELECT COUNT(*) FROM core.scanlationgroupfollow sgf
    WHERE sgf.groupid = sg.id
);
```

**POST /admin/batch/comics/counts** — Manual trigger for drift correction.

**Response `202 Accepted`:** `BatchJobRun` with `meta: { comicsUpdated: 52000, groupsUpdated: 840 }`.

---

### `announcements.expire` — Auto-hide Expired Announcements

**Trigger:** Scheduled  
**Tables:** `system.announcement`  
**Frequency:** Every hour at :05

**What it does:**
```sql
UPDATE system.announcement
SET ispublished = FALSE
WHERE ispublished = TRUE
  AND expiresat IS NOT NULL
  AND expiresat < NOW()
  AND deletedat IS NULL;
```

**POST /admin/batch/announcements/expire** — Manual trigger.

---

## 4. Analytics Batch Jobs

### `analytics.flush_counters` — Flush Redis View Counters to Postgres

**Trigger:** Scheduled  
**Tables:** `core.comic`, `core.chapter` (viewcount columns)  
**Frequency:** Every hour at :00

**What it does:**
```go
// 1. HGETALL all comic view counters from Redis
keys, _ := redis.Keys(ctx, "comic:*:views")
for _, key := range keys {
    count, _ := redis.GetDel(ctx, key)
    comicID  := extractIDFromKey(key)
    db.Exec("UPDATE core.comic SET viewcount = viewcount + $1 WHERE id = $2", count, comicID)
}

// 2. Same for chapters
keys, _ = redis.Keys(ctx, "chapter:*:views")
for _, key := range keys {
    count, _ := redis.GetDel(ctx, key)
    chapterID := extractIDFromKey(key)
    db.Exec("UPDATE core.chapter SET viewcount = viewcount + $1 WHERE id = $2", count, chapterID)
}
```

---

### POST /admin/batch/analytics/flush-counters

Force-flush Redis view counters to Postgres immediately  (useful before maintenance or shutdown).

**Auth required:** Yes (role: `admin`)

**Response `202 Accepted`:**
```json
{
  "data": {
    "jobkey": "analytics.flush_counters",
    "status": "queued",
    "meta": null
  }
}
```

On completion, `meta` contains:
```json
{ "comicsFlushed": 1240, "chaptersFlushed": 8420, "totalViewsWritten": 284021 }
```

---

### `analytics.anonymize` — Anonymize Old Pageview Data

**Trigger:** Scheduled  
**Tables:** `analytics.pageview`  
**Frequency:** Daily at 01:00 UTC

**What it does:**
```sql
-- Per privacy policy: anonymize IP and UA after 90 days
UPDATE analytics.pageview
SET ipaddress = NULL,
    useragent = NULL
WHERE createdat < NOW() - INTERVAL '90 days'
  AND (ipaddress IS NOT NULL OR useragent IS NOT NULL);
```

---

### POST /admin/batch/analytics/anonymize

Manually trigger IP/UA anonymization (e.g. after policy change).

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "olderthan": "90d" }
```

**Response `202 Accepted`:** `BatchJobRun` object.

On completion: `meta: { rowsAnonymized: 48200 }`.

---

### `analytics.partition` — Create Next Month's Analytics Partitions

**Trigger:** Scheduled  
**Tables:** `analytics.pageview`, `analytics.chaptersession`  
**Frequency:** 1st day of each month at 00:30 UTC

**What it does:**
```sql
-- Run on the 1st of each month to prepare next month's partitions:
CREATE TABLE IF NOT EXISTS analytics.pageview_YYYY_MM
    PARTITION OF analytics.pageview
    FOR VALUES FROM ('YYYY-MM-01') TO ('YYYY-MM+1-01');

CREATE TABLE IF NOT EXISTS analytics.chaptersession_YYYY_MM
    PARTITION OF analytics.chaptersession
    FOR VALUES FROM ('YYYY-MM-01') TO ('YYYY-MM+1-01');
```

**POST /admin/batch/analytics/partition** — Manual trigger (e.g. if scheduler missed).

**Response `202 Accepted`:** `BatchJobRun` with `meta: { partitionsCreated: ["pageview_2026_04", "chaptersession_2026_04"] }`.

---

### `crawler.partition` — Create Next Month's Crawler Log Partitions

**Trigger:** Scheduled  
**Tables:** `crawler.log`  
**Frequency:** 1st day of each month at 00:35 UTC

**POST /admin/batch/crawler/partitions** — Manual trigger.

**Response `202 Accepted`:** `BatchJobRun` with `meta: { partitionsCreated: ["log_2026_04"] }`.

---

## 5. System Maintenance Batch Jobs

### `sessions.cleanup` — Expire Stale Sessions

**Trigger:** Scheduled  
**Tables:** `users.session`  
**Frequency:** Every 6 hours

**What it does:**
```sql
DELETE FROM users.session
WHERE expiresat < NOW()
   OR revokedat IS NOT NULL;
```

---

### POST /admin/batch/sessions/cleanup

Manually trigger session cleanup (useful after a security incident).

**Auth required:** Yes (role: `admin`)

**Response `202 Accepted`:** `BatchJobRun` with `meta: { sessionsDeleted: 4821 }`.

---

### `crawler.source_health` — Auto-disable Failing Sources

**Trigger:** Triggered after every failed crawler job (not a scheduled sweep)  
**Tables:** `crawler.source`

**What it does:**
```go
// After each failed crawler job:
source.ConsecutiveFails++
if source.ConsecutiveFails >= setting("crawler.disable_threshold") {
    source.IsEnabled = false
    // Send admin notification
}
db.Save(source)
```

_(Documented in CRAWLER_API.md — listed here for completeness.)_

---

## 6. Admin — Batch Control Endpoints

### GET /admin/batch/jobs

List all batch job runs across all job types.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `jobkey` | string | — | Filter by job type, e.g. `library.hasnew` |
| `status` | string | — | `queued` \| `running` \| `done` \| `failed` \| `cancelled` |
| `triggeredby` | string | — | `scheduler` \| `admin` |
| `from` | string | 7 days ago | ISO 8601 |
| `to` | string | `NOW()` | ISO 8601 |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01953100-...",
      "jobkey": "analytics.flush_counters",
      "status": "done",
      "triggeredby": "scheduler",
      "triggeredbyuserid": null,
      "startedat": "2026-02-22T00:00:00Z",
      "finishedat": "2026-02-22T00:00:04Z",
      "durationms": 4120,
      "rowsaffected": 9842,
      "lasterror": null,
      "meta": { "comicsFlushed": 1240, "chaptersFlushed": 8420 },
      "createdat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 2840, "page": 1, "limit": 50, "pages": 57 }
}
```

---

### GET /admin/batch/jobs/:jobKey

Get run history for a specific job type.

**Auth required:** Yes (role: `admin`)  
**Path params:** `jobKey` — e.g. `library.hasnew`, `analytics.flush_counters`

**Query params:** `page`, `limit`, `status`

**Response `200 OK`:** Same shape as `GET /admin/batch/jobs` filtered to one job type.

---

### POST /admin/batch/jobs/:jobKey/run

Generic endpoint to manually trigger any registered batch job by its key.

**Auth required:** Yes (role: `admin`)  
**Path params:** `jobKey` — must be a registered job key (see §7)

**Request body:**
```json
{ "params": {} }
```

| Field | Type | Notes |
|---|---|---|
| `params` | object | Job-specific parameters. Shape varies per job. See individual job sections above. |

**Response `202 Accepted`:** New `BatchJobRun` object (`status = 'queued'`).

**Errors:**
```json
{ "error": "Unknown job key", "code": "NOT_FOUND" }
{ "error": "This job is already running", "code": "CONFLICT" }
```

---

### POST /admin/batch/jobs/:jobKey/cancel

Cancel a queued or running batch job.

**Auth required:** Yes (role: `admin`)

**Response `200 OK`:**
```json
{ "data": { "id": "01953100-...", "status": "cancelled" } }
```

**Side effects:** Go context cancel signal sent to the running goroutine.

---

### GET /admin/batch/schedule

View the full registered job schedule.

**Auth required:** Yes (role: `admin`)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "jobkey": "library.hasnew",
      "description": "Recalculate hasnew flag for all library entries",
      "cron": "*/15 * * * *",
      "isenabled": true,
      "lastrun": { "id": "...", "status": "done", "finishedat": "2026-02-22T00:15:00Z", "durationms": 820 },
      "nextrunat": "2026-02-22T00:30:00Z"
    },
    {
      "jobkey": "analytics.flush_counters",
      "description": "Flush Redis view counters to Postgres",
      "cron": "0 * * * *",
      "isenabled": true,
      "lastrun": { "id": "...", "status": "done", "finishedat": "2026-02-22T00:00:04Z", "durationms": 4120 },
      "nextrunat": "2026-02-22T01:00:00Z"
    }
  ]
}
```

---

### PATCH /admin/batch/schedule/:jobKey

Enable/disable a scheduled job or change its cron expression.

**Auth required:** Yes (role: `admin`)  
**Path params:** `jobKey`

**Request body:**
```json
{ "isenabled": false, "cron": null }
```

| Field | Type | Notes |
|---|---|---|
| `isenabled` | bool | `false` = skip this job in scheduler |
| `cron` | string \| null | Valid cron expression (5-field). `null` = keep current. |

**Response `200 OK`:** Updated `ScheduleEntry` object.

**Side effects:** Scheduler hot-reloads without restart. `system.auditlog` written.

**Errors:**
```json
{ "error": "Invalid cron expression", "code": "VALIDATION_ERROR" }
```

---

## 7. Scheduled Job Registry

| Job Key | Cron | Frequency | Description | Tables Affected |
|---|---|---|---|---|
| `library.hasnew` | `*/15 * * * *` | Every 15 min | Recalculate `hasnew` flag | `library.entry` |
| `library.viewhistory_cap` | `0 2 * * *` | Daily 02:00 | Cap view history to 500/user | `library.viewhistory` |
| `analytics.flush_counters` | `0 * * * *` | Every hour | Flush Redis view counters to DB | `core.comic`, `core.chapter` |
| `analytics.anonymize` | `0 1 * * *` | Daily 01:00 | Anonymize IP/UA older than 90d | `analytics.pageview` |
| `analytics.partition` | `30 0 1 * *` | 1st of month | Create next month's partitions | `analytics.pageview`, `analytics.chaptersession` |
| `crawler.partition` | `35 0 1 * *` | 1st of month | Create next month's crawler log partitions | `crawler.log` |
| `sessions.cleanup` | `0 */6 * * *` | Every 6 hours | Delete expired/revoked sessions | `users.session` |
| `comics.ratings_recalc` | `0 * * * *` | Every hour | Recalculate Bayesian ratings | `core.comic` |
| `comics.counts_recalc` | `*/30 * * * *` | Every 30 min | Recalculate chaptercount/followcount | `core.comic`, `core.scanlationgroup` |
| `announcements.expire` | `5 * * * *` | Every hour :05 | Auto-hide expired announcements | `system.announcement` |
| `storage.orphan_cleanup` | `0 3 * * 0` | Weekly Sunday | Delete orphaned media files | `core.mediafile` |

---

## 8. Implementation Notes

### Job runner architecture

```go
// Go scheduler using robfig/cron or similar
scheduler := cron.New(cron.WithSeconds())

scheduler.AddFunc("*/15 * * * *", func() {
    ctx := context.Background()
    run := batchService.StartRun(ctx, "library.hasnew", "scheduler", nil)
    affected, err := libraryService.RecalcHasNew(ctx, nil, nil)
    batchService.FinishRun(ctx, run.ID, affected, err)
})

scheduler.Start()
```

### Job run tracking

All batch job runs are tracked in a **lightweight in-memory store + optional Postgres table** (`system.batchrun` — can be added as a future migration). For v1.0.0, job history is kept in Redis as a sorted set (`ZADD batch:runs:{jobkey} timestamp runJSON`) with a retention of 30 days.

### Concurrency guard

```go
// Prevent duplicate runs of the same job
func (s *BatchService) StartRun(ctx context.Context, jobKey string, ...) (*BatchJobRun, error) {
    // Try to SET NX with TTL in Redis
    ok, _ := s.redis.SetNX(ctx, "batch:lock:"+jobKey, runID, 30*time.Minute)
    if !ok {
        return nil, ErrJobAlreadyRunning
    }
    // ... create run record
}

func (s *BatchService) FinishRun(ctx context.Context, runID string, ...) {
    s.redis.Del(ctx, "batch:lock:"+jobKey)
    // ... update run record
}
```

### Error handling & alerting

```go
// After a job fails 3 consecutive times, send admin notification
if consecutiveFailures >= 3 {
    notificationService.SendSystemAlert(ctx, SystemAlert{
        Level:   "critical",
        Message: fmt.Sprintf("Batch job %s has failed %d times consecutively", jobKey, consecutiveFailures),
    })
}
```
