# API Reference — Library Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `30_LIBRARY/LIBRARY.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).  
> All Library endpoints require authentication. Unauthenticated requests return `401 UNAUTHORIZED`.

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Library Entry, Custom Lists, Reading Progress, Chapter Read, View History. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Library Entry (My Shelf)](#2-library-entry-my-shelf)
3. [Custom Lists](#3-custom-lists)
4. [Reading Progress](#4-reading-progress)
5. [Chapter Read History](#5-chapter-read-history)
6. [View History](#6-view-history)
7. [Implementation Notes](#7-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/me/library` | Yes | Get my shelf (all library entries) |
| `GET` | `/me/library/:comicId` | Yes | Get a single library entry for a comic |
| `POST` | `/me/library/:comicId` | Yes | Add comic to shelf / create library entry |
| `PATCH` | `/me/library/:comicId` | Yes | Update reading status, score, notes |
| `DELETE` | `/me/library/:comicId` | Yes | Remove comic from shelf |
| `GET` | `/me/lists` | Yes | Get my custom lists |
| `POST` | `/me/lists` | Yes | Create a custom list |
| `GET` | `/lists/:id` | No* | Get a public/unlisted custom list |
| `PATCH` | `/me/lists/:id` | Yes | Update custom list metadata |
| `DELETE` | `/me/lists/:id` | Yes | Soft-delete a custom list |
| `GET` | `/me/lists/:id/items` | Yes | Get items in my custom list |
| `GET` | `/lists/:id/items` | No* | Get items in a public list |
| `POST` | `/me/lists/:id/items` | Yes | Add a comic to a custom list |
| `PATCH` | `/me/lists/:id/items` | Yes | Reorder items in a custom list |
| `DELETE` | `/me/lists/:id/items/:comicId` | Yes | Remove a comic from a custom list |
| `GET` | `/me/progress` | Yes | Get all reading progress records |
| `GET` | `/me/progress/:comicId` | Yes | Get reading progress for a specific comic |
| `PUT` | `/me/progress/:comicId` | Yes | Upsert reading progress (chapter + page) |
| `DELETE` | `/me/progress/:comicId` | Yes | Reset reading progress for a comic |
| `GET` | `/me/chapters/read` | Yes | List read chapters (optionally by comic) |
| `POST` | `/me/chapters/:chapterId/read` | Yes | Mark a chapter as read |
| `DELETE` | `/me/chapters/:chapterId/read` | Yes | Unmark a chapter as read |
| `POST` | `/me/comics/:comicId/read-all` | Yes | Mark all chapters of a comic as read |
| `DELETE` | `/me/comics/:comicId/read-all` | Yes | Unmark all chapters of a comic as read |
| `GET` | `/me/history` | Yes | Get comic view history |
| `DELETE` | `/me/history/:comicId` | Yes | Remove a comic from view history |
| `DELETE` | `/me/history` | Yes | Clear entire view history |

> `No*` = public lists accessible without auth; private/unlisted require ownership.

---

## 1. Common Types

### `LibraryEntry`
```typescript
{
  id: number
  userid: string
  comicid: string
  comic: ComicSummary          // from core domain
  readingstatus: "reading" | "completed" | "on_hold" | "dropped" | "plan_to_read"
  score: number | null         // private 1–10, distinct from public social.comicrating
  hasnew: boolean              // true = unread chapters available since last visit
  notes: string | null         // private freeform notes, visible only to owner
  lastreadchapterid: string | null
  lastreadchapter: {
    id: string
    chapternumber: number
    title: string | null
    language: Language
  } | null
  lastreadat: string | null
  createdat: string
  updatedat: string
}
```

### `CustomList`
```typescript
{
  id: string                   // UUIDv7 — used in shareable URLs: /list/{id}
  userid: string
  name: string
  description: string | null
  visibility: "public" | "private" | "unlisted"
  itemcount: number
  createdat: string
  updatedat: string
}
```

### `CustomListItem`
```typescript
{
  listid: string
  comic: ComicSummary
  sortorder: number
  createdat: string
}
```

### `ReadingProgress`
```typescript
{
  id: number
  userid: string
  comicid: string
  chapterid: string
  chapter: {
    id: string
    chapternumber: number
    title: string | null
    language: Language
    volume: number | null
  }
  pagenumber: number           // last page reached; 1 = chapter opened
  createdat: string
  updatedat: string
}
```

### `ViewHistoryItem`
```typescript
{
  id: number
  comicid: string
  comic: ComicSummary
  viewedat: string
}
```

---

## 2. Library Entry (My Shelf)

### GET /me/library

Get the current user's full library shelf with filtering and sorting.

**Auth required:** Yes  
**Rate limit:** 60 requests/min

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `status` | string | — | `reading` \| `completed` \| `on_hold` \| `dropped` \| `plan_to_read`. Multi-value allowed. |
| `hasnew` | bool | — | `true` = only entries with unread chapters |
| `sort` | string | `updatedat` | `updatedat` (desc) \| `createdat` (desc) \| `score` (desc) \| `title` (asc) \| `lastreadat` (desc) |
| `page` | int | `1` | — |
| `limit` | int | `24` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 1000001,
      "userid": "01952fa3-...",
      "comicid": "01952fb0-...",
      "comic": {
        "id": "01952fb0-...", "title": "Solo Leveling", "slug": "solo-leveling",
        "coverurl": "https://cdn.yomira.app/covers/solo-leveling.webp",
        "status": "completed", "contentrating": "safe", "chaptercount": 179
      },
      "readingstatus": "reading",
      "score": 9,
      "hasnew": true,
      "notes": null,
      "lastreadchapterid": "01952fa5-...",
      "lastreadchapter": {
        "id": "01952fa5-...", "chapternumber": 120.0,
        "title": null, "language": { "code": "en", "name": "English" }
      },
      "lastreadat": "2026-02-22T00:00:00Z",
      "createdat": "2026-02-01T00:00:00Z",
      "updatedat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 87, "page": 1, "limit": 24, "pages": 4 }
}
```

---

### GET /me/library/:comicId

Get the library entry for a single comic.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Response `200 OK`:** Single `LibraryEntry` object.

**Errors:**
```json
{ "error": "Entry not found", "code": "NOT_FOUND" }
```

---

### POST /me/library/:comicId

Add a comic to the shelf. Creates a `library.entry` row.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Request body:**
```json
{
  "readingstatus": "plan_to_read",
  "score": null,
  "notes": null
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `readingstatus` | string | No | `reading` \| `completed` \| `on_hold` \| `dropped` \| `plan_to_read`. Default: `plan_to_read` |
| `score` | int \| null | No | 1–10 (Go-validated). Private score, distinct from `social.comicrating`. |
| `notes` | string \| null | No | Private freeform notes, max 5000 chars |

**Response `201 Created`:** Full `LibraryEntry` object.

**Side effects:**
- `library.entry` row created
- `core.comic.followcount` incremented
- `social.feedevent` created (type: `library_add`) if status = `reading`

**Errors:**
```json
{ "error": "Comic already in your library", "code": "CONFLICT" }
{ "error": "Comic not found", "code": "NOT_FOUND" }
```

---

### PATCH /me/library/:comicId

Update reading status, personal score, or private notes.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Request body:** All fields optional (partial update).
```json
{
  "readingstatus": "completed",
  "score": 10,
  "notes": "Best manhwa I have ever read."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `readingstatus` | string | No | One of 5 allowed values |
| `score` | int \| null | No | 1–10 or `null` to clear |
| `notes` | string \| null | No | Max 5000 chars, or `null` to clear |

**Response `200 OK`:** Updated `LibraryEntry` object.

**Side effects:**
- `library.entry` updated
- If `readingstatus` changes to `completed`: `library.entry.score` may be prompted by the UI (no server-side enforcement)

**Errors:**
```json
{ "error": "Entry not found", "code": "NOT_FOUND" }
```

---

### DELETE /me/library/:comicId

Remove a comic from the shelf entirely.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Response `204 No Content`**

**Side effects:**
- `library.entry` row hard-deleted (cascade deletes `library.readingprogress`)
- `core.comic.followcount` decremented
- `library.chapterread` rows for this comic are **not** deleted (reading history preserved)

**Errors:**
```json
{ "error": "Entry not found", "code": "NOT_FOUND" }
```

---

## 3. Custom Lists

### GET /me/lists

Get all custom lists owned by the current user (including private).

**Auth required:** Yes  
**Query params:** `page`, `limit` (default 20, max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fc0-...", "userid": "01952fa3-...",
      "name": "My Top Isekai", "description": "The best isekai comics",
      "visibility": "public", "itemcount": 12,
      "createdat": "2026-02-01T00:00:00Z", "updatedat": "2026-02-22T00:00:00Z"
    },
    {
      "id": "01952fc1-...", "userid": "01952fa3-...",
      "name": "To Read Later", "description": null,
      "visibility": "private", "itemcount": 34,
      "createdat": "2026-01-15T00:00:00Z", "updatedat": "2026-01-20T00:00:00Z"
    }
  ],
  "meta": { "total": 5, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### POST /me/lists

Create a new custom list.

**Auth required:** Yes

**Request body:**
```json
{
  "name": "My Top Isekai",
  "description": "The best isekai comics I have read.",
  "visibility": "public"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | Yes | Max 200 chars |
| `description` | string \| null | No | Max 2000 chars |
| `visibility` | string | No | `public` \| `private` \| `unlisted`. Default: `private` |

**Response `201 Created`:** Full `CustomList` object.

**Side effects:** `library.customlist` row created (UUIDv7 generated).

**Errors:**
```json
{ "error": "You have reached the maximum number of custom lists (100)", "code": "LIMIT_EXCEEDED" }
```

---

### GET /lists/:id

Get a custom list by ID. Public and unlisted lists are accessible without auth. Private lists require ownership.

**Auth required:** No (for public/unlisted) | Yes (for private — must be owner)  
**Path params:** `id` — UUIDv7

**Response `200 OK`:** Full `CustomList` object (without items — use `GET /lists/:id/items`).

**Errors:**
```json
{ "error": "List not found", "code": "NOT_FOUND" }
{ "error": "This list is private", "code": "FORBIDDEN" }
```

---

### PATCH /me/lists/:id

Update a custom list's name, description, or visibility.

**Auth required:** Yes (owner only)  
**Path params:** `id` — UUIDv7

**Request body:** All fields optional (partial update).
```json
{
  "name": "My Absolute Favourites",
  "description": "Updated description.",
  "visibility": "unlisted"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | No | Max 200 chars |
| `description` | string \| null | No | Max 2000 chars, or `null` to clear |
| `visibility` | string | No | `public` \| `private` \| `unlisted` |

**Response `200 OK`:** Updated `CustomList` object.

**Errors:**
```json
{ "error": "List not found or you do not own this list", "code": "FORBIDDEN" }
```

---

### DELETE /me/lists/:id

Soft-delete a custom list.

**Auth required:** Yes (owner only)  
**Path params:** `id` — UUIDv7

**Response `204 No Content`**

**Side effects:** `library.customlist.deletedat = NOW()`. All `library.customlistitem` rows cascade-deleted.

---

### GET /me/lists/:id/items

Get items (comics) in one of the current user's lists.

**Auth required:** Yes  
**Path params:** `id` — UUIDv7  
**Query params:** `page`, `limit` (default 24, max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "listid": "01952fc0-...",
      "comic": {
        "id": "01952fb0-...", "title": "Solo Leveling", "slug": "solo-leveling",
        "coverurl": "...", "status": "completed"
      },
      "sortorder": 0,
      "createdat": "2026-02-10T00:00:00Z"
    }
  ],
  "meta": { "total": 12, "page": 1, "limit": 24, "pages": 1 }
}
```

---

### GET /lists/:id/items

Get items in a public or unlisted custom list. Accessible without auth.

**Auth required:** No (public/unlisted) | Yes (private — owner only)  
**Path params:** `id` — UUIDv7  
**Query params:** `page`, `limit`

**Response `200 OK`:** Same as `GET /me/lists/:id/items`.

**Errors:**
```json
{ "error": "List not found", "code": "NOT_FOUND" }
{ "error": "This list is private", "code": "FORBIDDEN" }
```

---

### POST /me/lists/:id/items

Add a comic to a custom list.

**Auth required:** Yes (owner only)  
**Path params:** `id` — UUIDv7

**Request body:**
```json
{
  "comicid": "01952fb0-...",
  "sortorder": 0
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `comicid` | string | Yes | Must exist in `core.comic`, not deleted |
| `sortorder` | int | No | Default `0`. Lower = shown first. |

**Response `201 Created`:** New `CustomListItem` object.

**Side effects:** `library.customlistitem` row inserted.

**Errors:**
```json
{ "error": "Comic is already in this list", "code": "CONFLICT" }
{ "error": "Comic not found", "code": "NOT_FOUND" }
{ "error": "List not found or you do not own this list", "code": "FORBIDDEN" }
{ "error": "List has reached the maximum item limit (500)", "code": "LIMIT_EXCEEDED" }
```

---

### PATCH /me/lists/:id/items

Reorder items in a custom list (bulk update of `sortorder`).

**Auth required:** Yes (owner only)  
**Path params:** `id` — UUIDv7

**Request body:**
```json
{
  "order": [
    { "comicid": "01952fb0-...", "sortorder": 0 },
    { "comicid": "01952fb1-...", "sortorder": 1 },
    { "comicid": "01952fb2-...", "sortorder": 2 }
  ]
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `order` | array | Yes | Max 500 items. Each item: `comicid` (string) + `sortorder` (int ≥ 0) |

**Response `200 OK`:**
```json
{ "data": { "updated": 3 } }
```

**Side effects:** `library.customlistitem.sortorder` batch-updated (`UPDATE ... WHERE listid = $1 AND comicid = $2`).

---

### DELETE /me/lists/:id/items/:comicId

Remove a specific comic from a custom list.

**Auth required:** Yes (owner only)  
**Path params:** `id` — list UUIDv7, `comicId` — comic UUIDv7

**Response `204 No Content`**

**Side effects:** `library.customlistitem` row hard-deleted.

**Errors:**
```json
{ "error": "Comic not in this list", "code": "NOT_FOUND" }
{ "error": "List not found or you do not own this list", "code": "FORBIDDEN" }
```

---

## 4. Reading Progress

Reading progress tracks the **last chapter and page** the user reached per comic. There is exactly one record per `(userid, comicid)` pair.

> **Note:** `POST /chapters/:id/read` (in CORE domain) automatically upserts reading progress. Use `PUT /me/progress/:comicId` only for explicit user-driven position saves (e.g. reader page sync).

### GET /me/progress

Get all reading progress records for the current user.

**Auth required:** Yes  
**Query params:** `page`, `limit` (default 24, max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 10001,
      "comicid": "01952fb0-...",
      "comic": { "id": "...", "title": "Solo Leveling", "coverurl": "..." },
      "chapterid": "01952fa5-...",
      "chapter": {
        "id": "01952fa5-...", "chapternumber": 120.0,
        "title": null, "volume": 12,
        "language": { "code": "en", "name": "English" }
      },
      "pagenumber": 8,
      "updatedat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 42, "page": 1, "limit": 24, "pages": 2 }
}
```

---

### GET /me/progress/:comicId

Get reading progress for a specific comic.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Response `200 OK`:** Single `ReadingProgress` object.

**Errors:**
```json
{ "error": "No reading progress found for this comic", "code": "NOT_FOUND" }
```

---

### PUT /me/progress/:comicId

Upsert reading progress — save the user's current chapter and page position. Called by the reader on every significant page change (debounced client-side).

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Request body:**
```json
{
  "chapterid": "01952fa5-...",
  "pagenumber": 8
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `chapterid` | string | Yes | Must belong to the specified `comicId` |
| `pagenumber` | int | Yes | ≥ 1. Must not exceed `core.chapter.pagecount` |

**Response `200 OK`:** Updated `ReadingProgress` object.

**Side effects:**
- `library.readingprogress` upserted (`INSERT ... ON CONFLICT (userid, comicid) DO UPDATE`)
- `library.entry.lastreadchapterid` + `lastreadat` updated (denormalized fast path)
- `library.entry.hasnew` recalculated: if `chapterid` is the latest chapter → `hasnew = FALSE`

**Errors:**
```json
{ "error": "Chapter does not belong to this comic", "code": "VALIDATION_ERROR" }
{ "error": "Page number exceeds chapter page count", "code": "VALIDATION_ERROR" }
```

---

### DELETE /me/progress/:comicId

Reset reading progress for a comic (user wants to start over).

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Response `204 No Content`**

**Side effects:**
- `library.readingprogress` row hard-deleted
- `library.entry.lastreadchapterid` set to `NULL`
- `library.entry.lastreadat` set to `NULL`
- `library.entry.hasnew` recalculated

---

## 5. Chapter Read History

### GET /me/chapters/read

List chapters the user has fully read.

**Auth required:** Yes

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `comicid` | string | — | Filter by comic UUIDv7 |
| `since` | string | — | ISO 8601 — chapters read after this time |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `500` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 20001,
      "chapterid": "01952fa5-...",
      "chapter": {
        "id": "01952fa5-...", "comicid": "01952fb0-...",
        "chapternumber": 120.0, "title": null,
        "language": { "code": "en" }
      },
      "readat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 120, "page": 1, "limit": 50, "pages": 3 }
}
```

---

### POST /me/chapters/:chapterId/read

Mark a specific chapter as fully read.

**Auth required:** Yes  
**Path params:** `chapterId` — UUIDv7

**Request body:** None

**Response `204 No Content`**

**Side effects:**
- `library.chapterread` inserted (`ON CONFLICT (userid, chapterid) DO NOTHING`)
- `library.readingprogress` upserted to this chapter's last page
- `library.entry.lastreadchapterid` + `lastreadat` updated
- `library.entry.hasnew` recalculated

---

### DELETE /me/chapters/:chapterId/read

Unmark a chapter as read (remove read record).

**Auth required:** Yes  
**Path params:** `chapterId` — UUIDv7

**Response `204 No Content`**

**Side effects:** `library.chapterread` row hard-deleted.

**Errors:**
```json
{ "error": "Chapter was not marked as read", "code": "NOT_FOUND" }
```

---

### POST /me/comics/:comicId/read-all

Mark **all chapters** of a comic as read in bulk.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Request body:**
```json
{ "language": "en" }
```

| Field | Type | Required | Description |
|---|---|---|---|
| `language` | string | No | BCP-47 code. If provided, only marks chapters in this language. Omit to mark all languages. |

**Response `200 OK`:**
```json
{ "data": { "markedcount": 179 } }
```

**Side effects:**
- `library.chapterread` bulk-inserted for all matching chapters (`ON CONFLICT DO NOTHING`)
- `library.readingprogress` upserted to the latest chapter
- `library.entry.hasnew = FALSE`
- `library.entry.lastreadchapterid` updated to last chapter

---

### DELETE /me/comics/:comicId/read-all

Unmark all chapters of a comic as read.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Request body:**
```json
{ "language": "en" }
```

| Field | Type | Required | Description |
|---|---|---|---|
| `language` | string | No | BCP-47 code. If provided, only unmarks chapters in this language. |

**Response `200 OK`:**
```json
{ "data": { "removedcount": 120 } }
```

**Side effects:** `library.chapterread` bulk-deleted for all matching chapters.

---

## 6. View History

View history records when a user **visits a comic detail page**. Distinct from reading progress (chapter-level). Capped at **500 entries per user** by background cleanup.

### GET /me/history

Get the current user's comic view history, newest first.

**Auth required:** Yes  
**Query params:** `page`, `limit` (default 24, max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 30001,
      "comicid": "01952fb0-...",
      "comic": {
        "id": "01952fb0-...", "title": "Solo Leveling", "slug": "solo-leveling",
        "coverurl": "https://cdn.yomira.app/covers/solo-leveling.webp",
        "status": "completed", "chaptercount": 179
      },
      "viewedat": "2026-02-22T00:09:59Z"
    },
    {
      "id": 30000,
      "comicid": "01952fa3-...",
      "comic": {
        "id": "01952fa3-...", "title": "One Piece", "slug": "one-piece",
        "coverurl": "...", "status": "ongoing"
      },
      "viewedat": "2026-02-22T00:05:00Z"
    }
  ],
  "meta": { "total": 243, "page": 1, "limit": 24, "pages": 11 }
}
```

> View history is recorded automatically by `GET /comics/:id` (fire-and-forget, async). No explicit client call needed.

---

### DELETE /me/history/:comicId

Remove a specific comic from view history.

**Auth required:** Yes  
**Path params:** `comicId` — UUIDv7

**Response `204 No Content`**

**Side effects:** All `library.viewhistory` rows for `(userid, comicid)` hard-deleted.

**Errors:**
```json
{ "error": "Comic not found in history", "code": "NOT_FOUND" }
```

---

### DELETE /me/history

Clear the entire view history.

**Auth required:** Yes

**Request body:** None

**Response `204 No Content`**

**Side effects:** All `library.viewhistory` rows for the user hard-deleted.

---

## 7. Implementation Notes

### Go validation (NOT SQL)

```go
// library.entry
readingstatus: "reading" | "completed" | "on_hold" | "dropped" | "plan_to_read"
score:         1 <= score <= 10 (if not nil)
notes:         len(notes) <= 5000 (if not nil)

// library.customlist
visibility:    "public" | "private" | "unlisted"
name:          len(name) <= 200
description:   len(description) <= 2000 (if not nil)

// library.customlistitem
sortorder:     >= 0
max items per list: 500 (service-layer check before INSERT)
max lists per user: 100 (service-layer check before INSERT)

// library.readingprogress
pagenumber:    >= 1
pagenumber:    <= core.chapter.pagecount (fetched, then validated)
chapterid:     must belong to the specified comicid (JOIN check)
```

### Key SQL patterns

**Upsert reading progress:**
```sql
INSERT INTO library.readingprogress (userid, comicid, chapterid, pagenumber)
VALUES ($1, $2, $3, $4)
ON CONFLICT (userid, comicid)
DO UPDATE SET chapterid = EXCLUDED.chapterid,
              pagenumber = EXCLUDED.pagenumber,
              updatedat  = NOW();
```

**Bulk mark-all-read:**
```sql
INSERT INTO library.chapterread (userid, chapterid)
SELECT $1, c.id FROM core.chapter c
WHERE c.comicid = $2
  AND ($3::varchar IS NULL OR c.languageid = (SELECT id FROM core.language WHERE code = $3))
  AND c.deletedat IS NULL
ON CONFLICT (userid, chapterid) DO NOTHING;
```

**`hasnew` recalculation:**
```sql
-- After marking chapter as read, check if any unread chapters remain
UPDATE library.entry SET hasnew = (
    EXISTS (
        SELECT 1 FROM core.chapter ch
        WHERE ch.comicid = $1
          AND ch.deletedat IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM library.chapterread cr
              WHERE cr.userid = $2 AND cr.chapterid = ch.id
          )
    )
)
WHERE userid = $2 AND comicid = $1;
```

**View history cap (background job):**
```sql
-- Runs periodically; evicts oldest entries beyond 500 per user
DELETE FROM library.viewhistory
WHERE id IN (
    SELECT id FROM library.viewhistory
    WHERE userid = $1
    ORDER BY viewedat DESC
    OFFSET 500
);
```

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /me/library` | No cache (personalized) | — |
| `GET /me/progress` | No cache (personalized) | — |
| `GET /lists/:id` (public) | 2 min | `list:{id}` |
| `GET /lists/:id/items` (public) | 2 min | `list:{id}:items:p{n}` |

> Library data is inherently user-specific and mutable — avoid caching. Only public custom lists benefit from short-TTL caching.

### Auto-recorded events (no explicit client call needed)

| Trigger | Side effect |
|---|---|
| `GET /comics/:id` | `library.viewhistory` row inserted (async, fire-and-forget) |
| `GET /chapters/:id` | `analytics.pageview` logged; view counts incremented via Redis |
| `POST /chapters/:id/read` (CORE domain) | `library.chapterread`, `readingprogress`, `entry` all updated |
| New chapter uploaded to followed comic | `library.entry.hasnew = TRUE` for all followers (background job) |
