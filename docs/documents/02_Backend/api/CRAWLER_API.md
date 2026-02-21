# API Reference — Crawler Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `50_CRAWLER/CRAWLER.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).  
> **All Crawler endpoints require `admin` role.** This domain is internal infrastructure — not exposed to regular users.

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Sources, Comic Sources, Jobs, Logs. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Sources](#2-sources)
3. [Comic Sources](#3-comic-sources)
4. [Jobs](#4-jobs)
5. [Logs](#5-logs)
6. [Implementation Notes](#6-implementation-notes)

---

## Endpoint Summary

> All endpoints require `Authorization: Bearer <access_token>` with role `admin`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/crawler/sources` | List all registered crawler sources |
| `GET` | `/admin/crawler/sources/:id` | Get a source by ID |
| `POST` | `/admin/crawler/sources` | Register a new source |
| `PATCH` | `/admin/crawler/sources/:id` | Update source config or toggle enabled |
| `DELETE` | `/admin/crawler/sources/:id` | Deregister a source |
| `PATCH` | `/admin/crawler/sources/:id/enable` | Enable or disable a source |
| `GET` | `/admin/crawler/sources/:id/health` | Get source health stats |
| `GET` | `/admin/comics/:id/sources` | List all source mappings for a comic |
| `POST` | `/admin/comics/:id/sources` | Add a source mapping for a comic |
| `PATCH` | `/admin/comics/:id/sources/:sourceId` | Update a comic source mapping |
| `DELETE` | `/admin/comics/:id/sources/:sourceId` | Remove a comic source mapping |
| `GET` | `/admin/crawler/jobs` | List crawl jobs (with filters) |
| `GET` | `/admin/crawler/jobs/:id` | Get a specific job detail |
| `POST` | `/admin/crawler/jobs` | Trigger a manual crawl job |
| `PATCH` | `/admin/crawler/jobs/:id/cancel` | Cancel a queued or running job |
| `GET` | `/admin/crawler/jobs/:id/logs` | Get structured logs for a job |

---

## 1. Common Types

### `CrawlerSource`
```typescript
{
  id: number
  name: string
  slug: string
  baseurl: string
  extensionid: string | null        // reverse-domain plugin ID, e.g. "com.yomira.extension.nettruyen"
  config: object                    // JSONB — extension-specific; schema defined by Go extension
  isenabled: boolean
  lastsucceededat: string | null
  lastfailedat: string | null
  consecutivefails: number          // auto-disabled by Go when threshold exceeded
  createdat: string
  updatedat: string
}
```

### `ComicSource`
```typescript
{
  id: number
  comicid: string
  sourceid: number
  source: { id: number; name: string; slug: string; baseurl: string }
  sourceid_ext: string              // source site's own ID / slug for this comic
  sourceurl: string                 // full URL on the source site
  isactive: boolean
  lastcrawlat: string | null
  createdat: string
  updatedat: string
}
```

### `CrawlJob`
```typescript
{
  id: string                        // UUIDv7 — also used as log correlation ID
  sourceid: number
  source: { id: number; name: string; slug: string }
  comicid: string | null            // null = full-catalogue discovery crawl
  comic: ComicSummary | null
  status: "queued" | "running" | "done" | "failed" | "cancelled"
  scheduledat: string
  startedat: string | null
  finishedat: string | null
  pagescount: number                // chapters/pages successfully fetched
  errorcount: number
  lasterror: string | null
  triggeredby: string | null        // null = scheduler; non-null = admin user ID
  triggeredbyuser: { id: string; username: string } | null
  createdat: string
  updatedat: string
}
```

### `CrawlLog`
```typescript
{
  id: string
  jobid: string
  level: "debug" | "info" | "warn" | "error"
  message: string
  meta: object | null               // JSONB: URL, HTTP status, retry count, page number, etc.
  createdat: string
}
```

---

## 2. Sources

### GET /admin/crawler/sources

List all registered crawler sources.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `isenabled` | bool | — | `true` = active sources only; `false` = disabled only |
| `q` | string | — | Search by `name` |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 100000,
      "name": "NetTruyen",
      "slug": "nettruyen",
      "baseurl": "https://nettruyen.com",
      "extensionid": "com.yomira.extension.nettruyen",
      "config": { "ratelimit": 2, "proxy": null },
      "isenabled": true,
      "lastsucceededat": "2026-02-22T00:00:00Z",
      "lastfailedat": null,
      "consecutivefails": 0,
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 8, "page": 1, "limit": 50, "pages": 1 }
}
```

---

### GET /admin/crawler/sources/:id

Get a single source by integer ID.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — integer

**Response `200 OK`:** Full `CrawlerSource` object.

**Errors:** `404 NOT_FOUND`

---

### POST /admin/crawler/sources

Register a new crawler source (extension plugin).

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{
  "name": "MangaFire",
  "slug": "mangafire",
  "baseurl": "https://mangafire.to",
  "extensionid": "com.yomira.extension.mangafire",
  "config": {
    "ratelimit": 3,
    "proxy": null,
    "headers": { "Accept-Language": "en-US" }
  },
  "isenabled": true
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | Yes | Max 100 chars. Must be unique. |
| `slug` | string | Yes | Max 120 chars. Lowercase alphanumeric + hyphens. Must be unique. |
| `baseurl` | string | Yes | Valid HTTPS URL |
| `extensionid` | string \| null | No | Reverse-domain format. Max 200 chars. |
| `config` | object | No | Arbitrary JSONB — extension-specific. Default `{}`. |
| `isenabled` | bool | No | Default `true` |

**Response `201 Created`:** Full `CrawlerSource` object.

**Side effects:** `crawler.source` row created. `system.auditlog` written.

**Errors:**
```json
{ "error": "A source with this name already exists", "code": "CONFLICT" }
{ "error": "A source with this slug already exists", "code": "CONFLICT" }
{ "error": "BaseURL must be a valid HTTPS URL", "code": "VALIDATION_ERROR" }
```

---

### PATCH /admin/crawler/sources/:id

Update source config, name, URLs, or toggle enabled state.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — integer

**Request body:** All fields optional (partial update).
```json
{
  "config": { "ratelimit": 1, "proxy": "http://proxy.internal:3128" },
  "isenabled": false
}
```

| Field | Type | Notes |
|---|---|---|
| `name` | string | Max 100 chars |
| `slug` | string | Max 120 chars |
| `baseurl` | string | Valid HTTPS URL |
| `extensionid` | string \| null | — |
| `config` | object | Merged with existing config — send full replacement object |
| `isenabled` | bool | — |

**Response `200 OK`:** Updated `CrawlerSource` object.

**Side effects:** `crawler.source` updated. `system.auditlog` written.

---

### DELETE /admin/crawler/sources/:id

Deregister a crawler source. All linked `crawler.comicsource` rows are cascade-deleted.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "reason": "Source site shut down permanently" }
```

**Response `204 No Content`**

**Side effects:** `crawler.source` row deleted (hard). `crawler.comicsource` rows cascade-deleted. `system.auditlog` written.

**Errors:**
```json
{ "error": "Cannot delete a source with running jobs. Cancel all jobs first.", "code": "CONFLICT" }
```

---

### PATCH /admin/crawler/sources/:id/enable

Quick toggle to enable or disable a source without a full PATCH.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "isenabled": false, "reason": "Site down for maintenance" }
```

| Field | Type | Required |
|---|---|---|
| `isenabled` | bool | Yes |
| `reason` | string | No |

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100000, "name": "NetTruyen",
    "isenabled": false, "updatedat": "2026-02-22T00:16:49Z"
  }
}
```

**Side effects:** `crawler.source.isenabled` updated. `consecutivefails` reset to `0` when re-enabling. `system.auditlog` written.

---

### GET /admin/crawler/sources/:id/health

Get health and runtime statistics for a source.

**Auth required:** Yes (role: `admin`)

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100000,
    "name": "NetTruyen",
    "isenabled": true,
    "consecutivefails": 0,
    "lastsucceededat": "2026-02-22T00:00:00Z",
    "lastfailedat": null,
    "jobs": {
      "total": 1024,
      "done": 1018,
      "failed": 4,
      "running": 2,
      "queued": 0
    },
    "lastjob": {
      "id": "01952ff9-...",
      "status": "done",
      "pagescount": 24,
      "errorcount": 0,
      "finishedat": "2026-02-22T00:00:00Z"
    }
  }
}
```

---

## 3. Comic Sources

A single comic can be linked to multiple source sites (e.g. both NetTruyen and MangaDex). Each `crawler.comicsource` row represents one `(comic, source)` pairing.

### GET /admin/comics/:id/sources

List all source mappings for a comic.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — comic UUIDv7

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 200001,
      "comicid": "01952fb0-...",
      "sourceid": 100000,
      "source": { "id": 100000, "name": "NetTruyen", "slug": "nettruyen", "baseurl": "https://nettruyen.com" },
      "sourceid_ext": "solo-leveling-chapter",
      "sourceurl": "https://nettruyen.com/truyen-tranh/solo-leveling-chapter",
      "isactive": true,
      "lastcrawlat": "2026-02-22T00:00:00Z",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-22T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/comics/:id/sources

Link a comic to a new source site.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — comic UUIDv7

**Request body:**
```json
{
  "sourceid": 100000,
  "sourceid_ext": "solo-leveling-chapter",
  "sourceurl": "https://nettruyen.com/truyen-tranh/solo-leveling-chapter",
  "isactive": true
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `sourceid` | int | Yes | Must exist in `crawler.source` |
| `sourceid_ext` | string | Yes | Source site's own ID/slug for this comic. Must be unique per `(sourceid, sourceid_ext)`. |
| `sourceurl` | string | Yes | Full URL on the source site. Valid HTTPS URL. |
| `isactive` | bool | No | Default `true` |

**Response `201 Created`:** New `ComicSource` object.

**Side effects:** `crawler.comicsource` row created.

**Errors:**
```json
{ "error": "This comic is already linked to the specified source with this external ID", "code": "CONFLICT" }
{ "error": "Source not found", "code": "NOT_FOUND" }
```

---

### PATCH /admin/comics/:id/sources/:sourceId

Update a comic-source mapping (e.g. source URL changed, toggle active).

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — comic UUIDv7, `sourceId` — `crawler.comicsource.id` integer

**Request body:** All optional.
```json
{
  "sourceurl": "https://nettruyen.net/truyen-tranh/solo-leveling",
  "sourceid_ext": "solo-leveling",
  "isactive": false
}
```

**Response `200 OK`:** Updated `ComicSource` object.

---

### DELETE /admin/comics/:id/sources/:sourceId

Remove a comic-source mapping.

**Auth required:** Yes (role: `admin`)

**Response `204 No Content`**

**Side effects:** `crawler.comicsource` row hard-deleted.

---

## 4. Jobs

A **job** is one crawl execution. `comicid = null` means a **full-catalogue discovery crawl** (finds new comics on the source). Non-null `comicid` = **targeted crawl** (fetches new chapters for a specific comic).

### GET /admin/crawler/jobs

List crawl jobs with filters.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `status` | string | — | `queued` \| `running` \| `done` \| `failed` \| `cancelled`. Multi-value allowed. |
| `sourceid` | int | — | Filter by source |
| `comicid` | string | — | Filter by comic (targeted jobs only) |
| `triggeredby` | string | — | `scheduler` (null triggeredby) \| `admin` (non-null) |
| `since` | string | — | ISO 8601 — jobs scheduled after this time |
| `sort` | string | `scheduledat` | `scheduledat` \| `finishedat` \| `pagescount` |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952ff9-...",
      "source": { "id": 100000, "name": "NetTruyen", "slug": "nettruyen" },
      "comicid": "01952fb0-...",
      "comic": { "id": "01952fb0-...", "title": "Solo Leveling", "coverurl": "..." },
      "status": "done",
      "scheduledat": "2026-02-22T00:00:00Z",
      "startedat": "2026-02-22T00:00:05Z",
      "finishedat": "2026-02-22T00:00:47Z",
      "pagescount": 24,
      "errorcount": 0,
      "lasterror": null,
      "triggeredby": null,
      "triggeredbyuser": null,
      "createdat": "2026-02-22T00:00:00Z",
      "updatedat": "2026-02-22T00:00:47Z"
    }
  ],
  "meta": { "total": 1024, "page": 1, "limit": 50, "pages": 21 }
}
```

---

### GET /admin/crawler/jobs/:id

Get a specific crawl job detail.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — job UUIDv7

**Response `200 OK`:** Full `CrawlJob` object.

**Errors:** `404 NOT_FOUND`

---

### POST /admin/crawler/jobs

Trigger a manual crawl job.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{
  "sourceid": 100000,
  "comicid": "01952fb0-...",
  "scheduledat": "2026-02-22T00:17:00Z"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `sourceid` | int | Yes | Must exist and be `isenabled = true` |
| `comicid` | string \| null | No | `null` = full-catalogue discovery crawl. Non-null = targeted comic crawl. |
| `scheduledat` | string | No | ISO 8601. Default: `NOW()` (immediate). |

**Response `201 Created`:** New `CrawlJob` object (`status = 'queued'`).

**Side effects:**
- `crawler.job` row created (`triggeredby = callerID`)
- Job placed in the Go crawler queue (Redis or in-memory)
- `system.auditlog` written

**Errors:**
```json
{ "error": "Source is disabled. Enable the source before triggering a job.", "code": "VALIDATION_ERROR" }
{ "error": "Comic is not linked to this source (no comicsource mapping found).", "code": "VALIDATION_ERROR" }
{ "error": "A job for this comic and source is already queued or running.", "code": "CONFLICT" }
```

---

### PATCH /admin/crawler/jobs/:id/cancel

Cancel a queued or running job.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — job UUIDv7

**Request body:**
```json
{ "reason": "Manual cancellation — source site temporarily unreachable" }
```

| Field | Type | Required |
|---|---|---|
| `reason` | string | No |

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01952ff9-...",
    "status": "cancelled",
    "updatedat": "2026-02-22T00:16:49Z"
  }
}
```

**Side effects:**
- `crawler.job.status = 'cancelled'`
- Go crawler receives cancellation signal (context cancel) if job is `running`
- `system.auditlog` written

**Errors:**
```json
{ "error": "Job cannot be cancelled — it has already finished", "code": "VALIDATION_ERROR" }
{ "error": "Job not found", "code": "NOT_FOUND" }
```

---

## 5. Logs

Crawl logs are **append-only** and **partitioned by month**. Do not filter by large date ranges — prefer filtering by `jobid`.

### GET /admin/crawler/jobs/:id/logs

Get structured log entries for a specific job.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — job UUIDv7

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `level` | string | — | `debug` \| `info` \| `warn` \| `error`. Filter by severity. |
| `since` | string | — | ISO 8601 cursor — logs after this time |
| `limit` | int | `100` | Max `1000` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952ffa-...",
      "jobid": "01952ff9-...",
      "level": "info",
      "message": "Starting targeted crawl for comic 01952fb0-...",
      "meta": { "comicid": "01952fb0-...", "sourceid": 100000 },
      "createdat": "2026-02-22T00:00:05Z"
    },
    {
      "id": "01952ffb-...",
      "jobid": "01952ff9-...",
      "level": "info",
      "message": "Found 2 new chapters",
      "meta": { "chapternumbers": [179, 180], "url": "https://nettruyen.com/truyen-tranh/solo-leveling" },
      "createdat": "2026-02-22T00:00:12Z"
    },
    {
      "id": "01952ffc-...",
      "jobid": "01952ff9-...",
      "level": "warn",
      "message": "Rate limit hit — retrying after 2s",
      "meta": { "url": "https://nettruyen.com/truyen-tranh/solo-leveling/chapter-180", "httpstatus": 429, "retryafter": 2 },
      "createdat": "2026-02-22T00:00:31Z"
    },
    {
      "id": "01952ffd-...",
      "jobid": "01952ff9-...",
      "level": "info",
      "message": "Job completed successfully",
      "meta": { "pagescount": 24, "durationms": 42000 },
      "createdat": "2026-02-22T00:00:47Z"
    }
  ],
  "meta": {
    "total": 87,
    "nextcursor": "2026-02-22T00:00:47Z"
  }
}
```

> Uses **cursor-based pagination** (`since` param) — log IDs are UUIDv7 monotonic, so clients can append without gaps.

---

## 6. Implementation Notes

### Go validation (NOT SQL)

```go
// crawler.source
name:       len(name) <= 100, must be unique
slug:       lowercase alphanumeric + hyphens (/^[a-z0-9-]+$/), max 120, must be unique
baseurl:    valid HTTPS URL
extensionid: reverse-domain format (optional)
config:     valid JSON object (arbitrary — extension-specific)

// crawler.comicsource
sourceid_ext: unique per (sourceid, sourceid_ext)
sourceurl:  valid HTTPS URL

// crawler.job
status transitions (Go FSM):
  queued → running → done | failed
  queued → cancelled
  running → cancelled (via context cancel signal)
  done / failed / cancelled → immutable (no further transitions)

// crawler.log
level: "debug" | "info" | "warn" | "error"
append-only — no UPDATE or DELETE ever issued
```

### Auto-disable source on consecutive failures

```go
// After each failed job, Go service runs:
UPDATE crawler.source
SET consecutivefails = consecutivefails + 1,
    lastfailedat     = NOW()
WHERE id = $sourceid;

// If consecutivefails >= threshold (e.g. 5):
UPDATE crawler.source
SET isenabled = FALSE
WHERE id = $sourceid AND consecutivefails >= 5;

// On successful job completion, reset:
UPDATE crawler.source
SET consecutivefails = 0,
    lastsucceededat  = NOW()
WHERE id = $sourceid;
```

### Partition management (crawler.log)

```sql
-- Create next month's partition (run monthly via pg_cron or Go job):
CREATE TABLE crawler.log_2026_07
    PARTITION OF crawler.log
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

-- Detach and archive partitions older than 6 months:
ALTER TABLE crawler.log DETACH PARTITION crawler.log_2026_01;
-- Move to cold storage or pg_dump, then DROP TABLE crawler.log_2026_01;
```

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /admin/crawler/sources` | 1 min | `crawler:sources:list` |
| `GET /admin/crawler/sources/:id` | 2 min | `crawler:source:{id}` |
| `GET /admin/crawler/sources/:id/health` | 30 sec | `crawler:source:{id}:health` |
| `GET /admin/crawler/jobs` | No cache (live data) | — |
| `GET /admin/crawler/jobs/:id/logs` | No cache (append-only, real-time) | — |

### Job queue architecture

```
Admin triggers POST /admin/crawler/jobs
        ↓
crawler.job row inserted (status = 'queued')
        ↓
Job ID pushed to Redis list: LPUSH crawler:queue {jobid}
        ↓
Go CrawlerWorker pools RPOP from crawler:queue
        ↓
Worker updates status = 'running', startedat = NOW()
        ↓
Worker fetches source config from crawler.source
Worker uses comicsource.sourceurl as starting point
Worker writes progress to crawler.log (append-only)
        ↓
On completion: status = 'done', pagescount, finishedat updated
On failure:    status = 'failed', lasterror saved
               source.consecutivefails incremented
```
