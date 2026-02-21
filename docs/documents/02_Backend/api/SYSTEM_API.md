# API Reference — System Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `70_SYSTEM/SYSTEM.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Audit Log, Settings, Announcements. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Audit Log](#2-audit-log)
3. [Settings](#3-settings)
4. [Announcements](#4-announcements)
5. [Implementation Notes](#5-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/admin/auditlog` | admin | List audit log entries with filters |
| `GET` | `/admin/auditlog/:id` | admin | Get a single audit log entry |
| `GET` | `/admin/settings` | admin | List all system settings |
| `GET` | `/admin/settings/:key` | admin | Get a single setting by key |
| `PUT` | `/admin/settings/:key` | admin | Create or update a setting |
| `DELETE` | `/admin/settings/:key` | admin | Delete a setting key |
| `GET` | `/announcements` | No | List published, non-expired announcements |
| `GET` | `/announcements/:id` | No | Get a single announcement |
| `GET` | `/admin/announcements` | admin/mod | List all announcements (including drafts) |
| `POST` | `/admin/announcements` | admin/mod | Create an announcement |
| `PATCH` | `/admin/announcements/:id` | admin/mod | Update an announcement |
| `PATCH` | `/admin/announcements/:id/publish` | admin/mod | Publish or unpublish an announcement |
| `DELETE` | `/admin/announcements/:id` | admin/mod | Soft-delete an announcement |

---

## 1. Common Types

### `AuditLog`
```typescript
{
  id: string                  // UUIDv7 — time-ordered
  actorid: string | null      // null if admin account was deleted (ON DELETE SET NULL)
  actor: { id: string; username: string; role: string } | null
  action: string              // dot-notation: "comic.delete" | "user.ban" | "chapter.lock" | ...
  entitytype: string | null   // "comic" | "chapter" | "user" | "group" | "comment" | ...
  entityid: string | null
  before: object | null       // JSON snapshot BEFORE action (null for create ops)
  after: object | null        // JSON snapshot AFTER action (null for delete ops)
  ipaddress: string | null
  createdat: string
}
```

### `Setting`
```typescript
{
  id: number
  key: string                 // dot-notation: "site.name" | "crawler.max_retries" | "auth.session_ttl_days"
  value: string | null        // always a string — Go coerces to correct type
  description: string | null
  createdat: string
  updatedat: string
}
```

### `Announcement`
```typescript
{
  id: string                  // UUIDv7
  authorid: string | null     // null if author account deleted
  author: { id: string; username: string } | null
  title: string
  body: string
  bodyformat: "markdown" | "html" | "plain"
  ispublished: boolean
  ispinned: boolean
  publishedat: string | null  // null = draft
  expiresat: string | null    // null = never expires
  createdat: string
  updatedat: string
}
```

---

## 2. Audit Log

The audit log is **append-only and immutable** — no UPDATE or DELETE is ever issued. It records all privileged administrative actions with before/after JSON snapshots for full accountability.

### GET /admin/auditlog

List audit log entries.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `actorid` | string | — | Filter by admin user ID |
| `action` | string | — | Exact or prefix match: `comic.` matches all comic actions |
| `entitytype` | string | — | `comic` \| `chapter` \| `user` \| `group` \| `comment` \| `forumpost` \| `report` \| … |
| `entityid` | string | — | Filter by specific entity |
| `from` | string | 30 days ago | ISO 8601 start |
| `to` | string | `NOW()` | ISO 8601 end |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `500` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01953000-...",
      "actorid": "01952fa3-...",
      "actor": { "id": "01952fa3-...", "username": "buivan", "role": "admin" },
      "action": "comic.delete",
      "entitytype": "comic",
      "entityid": "01952fb0-...",
      "before": { "id": "01952fb0-...", "title": "Solo Leveling", "status": "completed" },
      "after": null,
      "ipaddress": "203.0.113.42",
      "createdat": "2026-02-22T00:22:33Z"
    },
    {
      "id": "01953001-...",
      "actorid": "01952fa3-...",
      "actor": { "id": "01952fa3-...", "username": "buivan", "role": "admin" },
      "action": "user.ban",
      "entitytype": "user",
      "entityid": "01952fa9-...",
      "before": { "id": "01952fa9-...", "username": "spammer", "isactive": true },
      "after":  { "id": "01952fa9-...", "username": "spammer", "isactive": false },
      "ipaddress": "203.0.113.42",
      "createdat": "2026-02-22T00:20:00Z"
    }
  ],
  "meta": { "total": 4821, "page": 1, "limit": 50, "pages": 97 }
}
```

---

### GET /admin/auditlog/:id

Get a single audit log entry with full before/after JSON snapshots.

**Auth required:** Yes (role: `admin`)  
**Path params:** `id` — audit log UUIDv7

**Response `200 OK`:** Full `AuditLog` object.

**Errors:** `404 NOT_FOUND`

---

## 3. Settings

Global key-value configuration store. Loaded by Go at startup and **cached in memory**. Changes take effect on next Go service restart or explicit cache-flush signal.

> Key naming convention: **dot-notation namespacing**  
> Examples: `site.name`, `site.maintenance_mode`, `crawler.max_retries`, `auth.session_ttl_days`, `upload.max_file_size_mb`

### GET /admin/settings

List all system settings.

**Auth required:** Yes (role: `admin`)

**Query params:** `q` (search by key prefix), `page`, `limit` (default 100, max 500)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 100000,
      "key": "site.name",
      "value": "Yomira",
      "description": "The display name of the site shown in the browser title and emails.",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-01T00:00:00Z"
    },
    {
      "id": 100001,
      "key": "site.maintenance_mode",
      "value": "false",
      "description": "Set to true to show a maintenance page to all non-admin users.",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-01T00:00:00Z"
    },
    {
      "id": 100002,
      "key": "auth.session_ttl_days",
      "value": "30",
      "description": "Refresh token lifetime in days.",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-01T00:00:00Z"
    },
    {
      "id": 100003,
      "key": "crawler.max_retries",
      "value": "3",
      "description": "Maximum number of retry attempts per failed crawl request.",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-01T00:00:00Z"
    },
    {
      "id": 100004,
      "key": "upload.max_file_size_mb",
      "value": "10",
      "description": "Maximum file size (MB) for image uploads.",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-01T00:00:00Z"
    }
  ],
  "meta": { "total": 18, "page": 1, "limit": 100, "pages": 1 }
}
```

---

### GET /admin/settings/:key

Get a single setting by its dot-notation key.

**Auth required:** Yes (role: `admin`)  
**Path params:** `key` — e.g. `site.name`, `crawler.max_retries`

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100001,
    "key": "site.maintenance_mode",
    "value": "false",
    "description": "Set to true to show a maintenance page to all non-admin users.",
    "createdat": "2026-02-01T00:00:00Z",
    "updatedat": "2026-02-22T00:22:33Z"
  }
}
```

**Errors:** `404 NOT_FOUND`

---

### PUT /admin/settings/:key

Create or update a setting. Upserts on `key`.

**Auth required:** Yes (role: `admin`)  
**Path params:** `key` — dot-notation key

**Request body:**
```json
{
  "value": "true",
  "description": "Set to true to show a maintenance page to all non-admin users."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `value` | string \| null | Yes | Always stored as string. Go handles type coercion. Max 10 000 chars. |
| `description` | string \| null | No | Human-readable explanation for admin UI. |

**Response `200 OK`:** Updated `Setting` object.

**Side effects:**
- `system.setting` upserted (`INSERT ... ON CONFLICT (key) DO UPDATE`)
- In-memory config cache **invalidated** (Go sends internal signal to reload)
- `system.auditlog` row written (action: `setting.update`)

**Errors:**
```json
{ "error": "Key must follow dot-notation format (e.g. site.name)", "code": "VALIDATION_ERROR" }
```

---

### DELETE /admin/settings/:key

Delete a setting key.

**Auth required:** Yes (role: `admin`)  
**Path params:** `key` — dot-notation key

**Response `204 No Content`**

**Side effects:** `system.setting` row hard-deleted. In-memory config cache invalidated. `system.auditlog` written.

**Errors:**
```json
{ "error": "Setting not found", "code": "NOT_FOUND" }
```

---

## 4. Announcements

Announcements are site-wide notices published by admins or moderators. They appear on the Community / Announcements page. Soft-deleted via `deletedat`. Auto-hidden by Go after `expiresat`.

### GET /announcements

List active published announcements — **public endpoint**.

**Auth required:** No

**Query params:** `page`, `limit` (default 10, max 50)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01953010-...",
      "author": { "id": "01952fa3-...", "username": "buivan" },
      "title": "Yomira v2.0 is live!",
      "body": "We are excited to announce the launch of Yomira v2.0...",
      "bodyformat": "markdown",
      "ispublished": true,
      "ispinned": true,
      "publishedat": "2026-02-22T00:00:00Z",
      "expiresat": null,
      "createdat": "2026-02-21T22:00:00Z",
      "updatedat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 3, "page": 1, "limit": 10, "pages": 1 }
}
```

> **Filter applied:** `ispublished = TRUE AND deletedat IS NULL AND (expiresat IS NULL OR expiresat > NOW())`.  
> Pinned announcements (`ispinned = TRUE`) always appear first.

---

### GET /announcements/:id

Get a single published announcement.

**Auth required:** No  
**Path params:** `id` — announcement UUIDv7

**Response `200 OK`:** Full `Announcement` object.

**Errors:**
```json
{ "error": "Announcement not found", "code": "NOT_FOUND" }
```

---

### GET /admin/announcements

List **all** announcements including drafts and soft-deleted. Admin/mod view.

**Auth required:** Yes (role: `admin` | `moderator`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `ispublished` | bool | — | `true` = published only; `false` = drafts only |
| `includeexpired` | bool | `false` | Include announcements past `expiresat` |
| `includedeleted` | bool | `false` | Include soft-deleted announcements |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:** Same shape as public `GET /announcements` but includes drafts, expired, and deleted rows.

---

### POST /admin/announcements

Create a new announcement (starts as draft).

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{
  "title": "Scheduled Maintenance — Feb 23",
  "body": "Yomira will be offline for maintenance on **Feb 23 00:00–02:00 UTC**.",
  "bodyformat": "markdown",
  "ispinned": false,
  "expiresat": "2026-02-24T00:00:00Z"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `title` | string | Yes | Max 300 chars |
| `body` | string | Yes | Max 100 000 chars |
| `bodyformat` | string | No | `markdown` \| `html` \| `plain`. Default: `markdown` |
| `ispinned` | bool | No | Default `false`. Only admin can set `true`. |
| `expiresat` | string \| null | No | ISO 8601. Must be in the future. |

**Response `201 Created`:** New `Announcement` object (`ispublished = false`, `publishedat = null`).

**Side effects:** `system.announcement` row created. `system.auditlog` written.

**Errors:**
```json
{ "error": "expiresat must be in the future", "code": "VALIDATION_ERROR" }
{ "error": "Only admin can pin announcements", "code": "FORBIDDEN" }
```

---

### PATCH /admin/announcements/:id

Update announcement content (works on drafts and published).

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — announcement UUIDv7

**Request body:** All fields optional (partial update).
```json
{
  "title": "Scheduled Maintenance — Feb 23 (Updated)",
  "body": "Maintenance window extended to 04:00 UTC.",
  "expiresat": "2026-02-25T00:00:00Z"
}
```

| Field | Type | Notes |
|---|---|---|
| `title` | string | Max 300 chars |
| `body` | string | Max 100 000 chars |
| `bodyformat` | string | `markdown` \| `html` \| `plain` |
| `ispinned` | bool | Admin only |
| `expiresat` | string \| null | Must be future, or `null` to remove expiry |

**Response `200 OK`:** Updated `Announcement` object.

**Side effects:** `system.announcement` updated. `system.auditlog` written.

---

### PATCH /admin/announcements/:id/publish

Publish or unpublish an announcement.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — announcement UUIDv7

**Request body:**
```json
{ "ispublished": true }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `ispublished` | bool | Yes | `true` = publish (sets `publishedat = NOW()` if null); `false` = unpublish (clears `publishedat`) |

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01953010-...",
    "ispublished": true,
    "publishedat": "2026-02-22T00:22:33Z",
    "updatedat": "2026-02-22T00:22:33Z"
  }
}
```

**Side effects:**
- `system.announcement.ispublished` updated
- If publishing: `publishedat = NOW()` (only set once — republishing preserves original `publishedat`)
- `social.notification` created for all users (type: `announcement`) — async, batch
- `system.auditlog` written

**Errors:**
```json
{ "error": "Cannot publish a deleted announcement", "code": "VALIDATION_ERROR" }
```

---

### DELETE /admin/announcements/:id

Soft-delete an announcement.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — announcement UUIDv7

**Response `204 No Content`**

**Side effects:** `system.announcement.deletedat = NOW()`. Announcement immediately hidden from public endpoint. `system.auditlog` written.

---

## 5. Implementation Notes

### Go validation (NOT SQL)

```go
// system.setting
key:   must match /^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$/ — dot-notation format
value: len(value) <= 10000 (if not nil)

// system.announcement
title:      len(title) <= 300
body:       len(body) <= 100000
bodyformat: "markdown" | "html" | "plain"
expiresat:  must be > NOW() if provided
ispinned:   only admin role can set true (moderator always sets false)

// system.auditlog
append-only — never UPDATE or DELETE
before/after: JSON-serialized Go structs, not arbitrary maps
```

### Well-known setting keys

| Key | Type | Default | Description |
|---|---|---|---|
| `site.name` | string | `"Yomira"` | Site display name |
| `site.description` | string | `"..."` | SEO meta description |
| `site.maintenance_mode` | bool | `"false"` | Shows maintenance page to non-admins |
| `auth.session_ttl_days` | int | `"30"` | Refresh token lifetime |
| `auth.max_sessions_per_user` | int | `"10"` | Max active sessions per user |
| `upload.max_file_size_mb` | int | `"10"` | Max image upload size |
| `crawler.max_retries` | int | `"3"` | Max retries per crawl request |
| `crawler.disable_threshold` | int | `"5"` | Consecutive fails before auto-disable |
| `social.comment_min_age_days` | int | `"0"` | Account age required to comment |
| `social.new_user_comment_limit` | int | `"5"` | Comments before auto-approve kicks in |

### Audit log action names (dot-notation)

| Action | Trigger |
|---|---|
| `comic.create` | `POST /comics` |
| `comic.update` | `PATCH /comics/:id` |
| `comic.delete` | `DELETE /comics/:id` |
| `comic.lock` | `PATCH /admin/comics/:id/lock` |
| `chapter.lock` | `PATCH /admin/chapters/:id/lock` |
| `chapter.official` | `PATCH /admin/chapters/:id/official` |
| `user.role_change` | `PATCH /admin/users/:id/role` |
| `user.ban` | `PATCH /admin/users/:id/suspend` |
| `user.delete` | `DELETE /admin/users/:id` |
| `group.verify` | `PATCH /admin/groups/:id/verify` |
| `group.suspend` | `PATCH /admin/groups/:id/suspend` |
| `group.delete` | `DELETE /admin/groups/:id` |
| `report.resolve` | `PATCH /admin/reports/:id` |
| `setting.update` | `PUT /admin/settings/:key` |
| `setting.delete` | `DELETE /admin/settings/:key` |
| `announcement.publish` | `PATCH /admin/announcements/:id/publish` |
| `announcement.delete` | `DELETE /admin/announcements/:id` |

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /announcements` (public) | 5 min | `announcements:public:p{n}` |
| `GET /admin/settings` | App startup (in-memory) | Invalidated by `PUT/DELETE /admin/settings/*` |
| `GET /admin/auditlog` | No cache (live data) | — |

> Settings are loaded into a `sync.Map` at Go startup. `PUT /admin/settings/:key` broadcasts an in-process invalidation signal (or triggers a process restart in single-instance deployments).
