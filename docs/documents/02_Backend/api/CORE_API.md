# API Reference — Core Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-21  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `20_CORE/CORE.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-21 | Initial release. Languages, Authors, Artists, Tags, Scanlation Groups, Comics, Chapters, Pages. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Languages](#2-languages)
3. [Authors](#3-authors)
4. [Artists](#4-artists)
5. [Tag Groups & Tags](#5-tag-groups--tags)
6. [Scanlation Groups](#6-scanlation-groups)
7. [Comics](#7-comics)
8. [Comic Titles](#8-comic-titles)
9. [Comic Relations](#9-comic-relations)
10. [Comic Covers](#10-comic-covers)
11. [Comic Art Gallery](#11-comic-art-gallery)
12. [Chapters](#12-chapters)
13. [Pages](#13-pages)
14. [Implementation Notes](#14-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/languages` | No | List all languages |
| `GET` | `/languages/:code` | No | Get language by BCP-47 code |
| `GET` | `/authors` | No | Search / list authors |
| `GET` | `/authors/:id` | No | Author detail + comics list |
| `POST` | `/authors` | admin/mod | Create author |
| `PATCH` | `/authors/:id` | admin/mod | Update author |
| `DELETE` | `/authors/:id` | admin | Soft-delete author |
| `GET` | `/artists` | No | Search / list artists |
| `GET` | `/artists/:id` | No | Artist detail + comics list |
| `POST` | `/artists` | admin/mod | Create artist |
| `PATCH` | `/artists/:id` | admin/mod | Update artist |
| `DELETE` | `/artists/:id` | admin | Soft-delete artist |
| `GET` | `/tags` | No | List all tags grouped by tag group |
| `GET` | `/tags/:id` | No | Get tag by ID |
| `GET` | `/tags/by-slug/:slug` | No | Get tag by slug |
| `GET` | `/groups` | No | Search / list scanlation groups |
| `GET` | `/groups/:id` | No | Group detail |
| `GET` | `/groups/:id/chapters` | No | Chapters uploaded by this group |
| `GET` | `/groups/:id/members` | No | List group members |
| `POST` | `/groups` | member | Create scanlation group |
| `PATCH` | `/groups/:id` | leader/mod/admin | Update group info |
| `POST` | `/groups/:id/members` | leader/admin | Add member to group |
| `PATCH` | `/groups/:id/members/:userId/role` | leader/admin | Change member role |
| `DELETE` | `/groups/:id/members/:userId` | leader/self/admin | Remove member from group |
| `POST` | `/groups/:id/follow` | member | Follow a group |
| `DELETE` | `/groups/:id/follow` | member | Unfollow a group |
| `GET` | `/me/groups` | Yes | Groups I am a member of |
| `GET` | `/me/groups/following` | Yes | Groups I follow |
| `PATCH` | `/admin/groups/:id/verify` | admin | Grant/revoke official publisher badge |
| `PATCH` | `/admin/groups/:id/suspend` | admin | Suspend group (`isactive = FALSE`) |
| `DELETE` | `/admin/groups/:id` | admin | Soft-delete group |
| `GET` | `/comics` | No | List / filter / search comics |
| `GET` | `/comics/:id` | No | Comic detail (accepts UUID or slug) |
| `GET` | `/comics/:id/chapters` | No | Paginated chapter list for a comic |
| `POST` | `/comics` | admin/mod | Create comic |
| `PATCH` | `/comics/:id` | admin/mod | Update comic metadata |
| `DELETE` | `/comics/:id` | admin | Soft-delete comic |
| `PATCH` | `/admin/comics/:id/lock` | admin | Toggle moderation lock |
| `GET` | `/comics/:id/titles` | No | List all per-language titles |
| `PUT` | `/comics/:id/titles/:languageCode` | admin/mod | Upsert title for a language |
| `DELETE` | `/comics/:id/titles/:languageCode` | admin/mod | Remove title for a language |
| `GET` | `/comics/:id/relations` | No | List all comic relations |
| `POST` | `/comics/:id/relations` | admin/mod | Add a comic relation |
| `DELETE` | `/comics/:id/relations/:toComicId/:type` | admin/mod | Remove a comic relation |
| `GET` | `/comics/:id/covers` | No | List volume covers |
| `POST` | `/comics/:id/covers` | admin/mod | Upload a volume cover |
| `DELETE` | `/comics/:id/covers/:coverId` | admin/mod | Delete a cover |
| `GET` | `/comics/:id/art` | No | List art gallery images |
| `POST` | `/comics/:id/art` | member/admin/mod | Upload art image |
| `PATCH` | `/admin/comics/:id/art/:artId/approve` | admin/mod | Approve / reject fanart |
| `DELETE` | `/comics/:id/art/:artId` | uploader/admin/mod | Delete art image |
| `GET` | `/chapters/:id` | No | Chapter detail + full page list |
| `POST` | `/chapters` | group member/admin/mod | Upload new chapter |
| `POST` | `/chapters/:id/pages` | group member/admin/mod | Upload pages (batch) |
| `PATCH` | `/chapters/:id` | leader/mod/admin | Update chapter metadata |
| `DELETE` | `/chapters/:id` | leader/admin/mod | Soft-delete chapter |
| `POST` | `/chapters/:id/read` | member | Mark chapter as read |
| `GET` | `/chapters/:id/pages` | No | Standalone page list |
| `DELETE` | `/admin/chapters/:id/pages/:pageNum` | admin | Delete a specific page (DMCA) |
| `PATCH` | `/admin/chapters/:id/lock` | admin/mod | Toggle chapter lock |
| `PATCH` | `/admin/chapters/:id/official` | admin | Set chapter as official |
| `GET` | `/admin/chapters` | admin | List chapters by sync state |

---

## 1. Common Types

### `Language`
```typescript
{ id: number; code: string; name: string; nativename: string | null }
```

### `Author` / `Artist`
```typescript
{
  id: number; name: string; namealt: string[]
  bio: string | null; imageurl: string | null
  createdat: string; updatedat: string
}
```

### `TagGroup`
```typescript
{ id: number; name: string; slug: string; sortorder: number }
```

### `Tag`
```typescript
{
  id: number; groupid: number; group: TagGroup
  name: string; slug: string; description: string | null
}
```

### `ScanlationGroup`
```typescript
{
  id: string              // UUIDv7
  name: string; slug: string; description: string | null
  website: string | null; discord: string | null; twitter: string | null
  patreon: string | null; youtube: string | null; mangaupdates: string | null
  isofficialpublisher: boolean; isactive: boolean; isfocused: boolean
  verifiedat: string | null
  membercount: number; followcount: number
  createdat: string; updatedat: string
}
```

### `ScanlationGroupMember`
```typescript
{
  userid: string; username: string; displayname: string | null
  avatarurl: string | null
  role: "leader" | "moderator" | "member"
  joinedat: string
}
```

### `ComicSummary`
```typescript
{
  id: string; title: string; slug: string; coverurl: string | null
  status: "ongoing" | "completed" | "hiatus" | "cancelled" | "unknown"
  contentrating: "safe" | "suggestive" | "explicit"
  demographic: "shounen" | "shoujo" | "seinen" | "josei" | null
  originlanguage: string | null; year: number | null
  ratingbayesian: number; followcount: number; chaptercount: number
  tags: Tag[]
  latestchapter: { id: string; chapternumber: number; publishedat: string; language: Language } | null
}
```

### `ComicDetail`
```typescript
{
  // All ComicSummary fields, plus:
  bannerurl: string | null; synopsis: string | null
  defaultreadmode: "ltr" | "rtl" | "vertical" | "webtoon"
  links: { mal?: string; anilist?: string; official?: string; raw?: string }
  viewcount: number; ratingavg: number; ratingcount: number
  islocked: boolean
  authors: Author[]; artists: Artist[]
  relations: ComicRelation[]; covers: ComicCover[]
  createdat: string; updatedat: string
  // Auth-dependent (null if unauthenticated):
  isFollowing: boolean | null
  userRating: number | null
  readingStatus: string | null
  lastReadChapterNumber: number | null
}
```

### `ComicRelation`
```typescript
{
  relatedcomic: ComicSummary
  relationtype: "sequel" | "prequel" | "main_story" | "side_story" |
                "spin_off" | "adaptation" | "alternate" | "same_franchise" |
                "colored" | "preserialization"
}
```

### `ComicCover`
```typescript
{ id: string; volume: number | null; imageurl: string; description: string | null; createdat: string }
```

### `Chapter`
```typescript
{
  id: string; comicid: string
  volume: number | null; chapternumber: number    // NUMERIC supports 12.5
  title: string | null
  language: Language
  scanlationgroup: ScanlationGroup | null
  uploaderid: string | null
  syncstate: "pending" | "processing" | "synced" | "failed" | "missing"
  externalurl: string | null; isofficial: boolean; islocked: boolean
  pagecount: number; viewcount: number
  publishedat: string | null; createdat: string; updatedat: string
}
```

### `Page`
```typescript
{
  id: string; chapterid: string; pagenumber: number
  imageurl: string        // compressed / datasaver
  imageurlhd: string | null
  format: string; width: number | null; height: number | null; filesizebytes: number | null
}
```

---

## 2. Languages

### GET /languages

List all supported languages. **Static reference data.**

**Auth required:** No  
**Cache:** `Cache-Control: public, max-age=86400`

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 100001, "code": "en", "name": "English", "nativename": "English" },
    { "id": 100002, "code": "vi", "name": "Vietnamese", "nativename": "Tiếng Việt" },
    { "id": 100003, "code": "ja", "name": "Japanese", "nativename": "日本語" }
  ]
}
```

---

### GET /languages/:code

Get a single language by BCP-47 code.

**Auth required:** No  
**Path params:** `code` — e.g. `en`, `vi`, `zh-hans`

**Response `200 OK`:** Single `Language` object.

**Errors:**
```json
{ "error": "Language not found", "code": "NOT_FOUND" }
```

---

## 3. Authors

### GET /authors

Search and list authors (used for filter autocomplete).

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `name` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 100001, "name": "Eiichiro Oda", "namealt": ["尾田 栄一郎"], "imageurl": null, "bio": null }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### GET /authors/:id

Author detail + comics list.

**Auth required:** No  
**Path params:** `id` — integer  
**Query params:** `page`, `limit` for comic list

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100001,
    "name": "Eiichiro Oda",
    "namealt": ["尾田 栄一郎"],
    "bio": "Creator of One Piece.",
    "imageurl": null,
    "createdat": "2026-02-21T00:00:00Z",
    "updatedat": "2026-02-21T00:00:00Z",
    "comics": [
      { "id": "01952fa3-...", "title": "One Piece", "coverurl": "...", "status": "ongoing" }
    ],
    "comiccount": 1
  }
}
```

**Errors:** `404 NOT_FOUND`

---

### POST /authors

Create a new author.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{
  "name": "Eiichiro Oda",
  "namealt": ["尾田 栄一郎"],
  "bio": "Creator of One Piece.",
  "imageurl": null
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | Yes | Max 200 chars |
| `namealt` | string[] | No | Each max 200 chars |
| `bio` | string \| null | No | — |
| `imageurl` | string \| null | No | Valid HTTPS URL |

**Response `201 Created`:** Full `Author` object.

**Side effects:** `core.author` row created. `system.auditlog` written.

---

### PATCH /authors/:id

Update an author.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — integer

**Request body:** All fields optional (partial update) — same shape as `POST /authors`.

**Response `200 OK`:** Updated `Author` object.

---

### DELETE /authors/:id

Soft-delete an author.

**Auth required:** Yes (role: `admin`)

**Response `204 No Content`**

---

## 4. Artists

### GET /artists

Search and list artists (used for filter autocomplete).

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `name` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 100010, "name": "Kim Jung-hyun", "namealt": ["김정현"], "imageurl": null, "bio": null }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### GET /artists/:id

Artist detail with list of comics they illustrated.

**Auth required:** No  
**Path params:** `id` — integer  
**Query params:** `page`, `limit` for comic list

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100010,
    "name": "Kim Jung-hyun",
    "namealt": ["김정현"],
    "bio": "Illustrator of Solo Leveling.",
    "imageurl": null,
    "createdat": "2026-02-22T00:02:08Z",
    "updatedat": "2026-02-22T00:02:08Z",
    "comics": [
      { "id": "01952fb0-...", "title": "Solo Leveling", "coverurl": "...", "status": "completed" }
    ],
    "comiccount": 1
  }
}
```

**Errors:** `404 NOT_FOUND`

---

### POST /artists

Create a new artist.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{
  "name": "Kim Jung-hyun",
  "namealt": ["김정현"],
  "bio": "Illustrator of Solo Leveling.",
  "imageurl": null
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | Yes | Max 200 chars |
| `namealt` | string[] | No | Each max 200 chars |
| `bio` | string \| null | No | — |
| `imageurl` | string \| null | No | Valid HTTPS URL |

**Response `201 Created`:** Full `Artist` object.

**Side effects:** `core.artist` row created. `system.auditlog` written.

---

### PATCH /artists/:id

Update an artist's information.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — integer

**Request body:** All fields optional (partial update) — same shape as `POST /artists`.

**Response `200 OK`:** Updated `Artist` object.

---

### DELETE /artists/:id

Soft-delete an artist.

**Auth required:** Yes (role: `admin`)

**Response `204 No Content`**

**Side effects:** `core.artist.deletedat = NOW()`. Existing comic links preserved. `system.auditlog` written.

---

## 5. Tag Groups & Tags

### GET /tags

List all tags grouped by tag group. **Static data — cacheable.**

**Auth required:** No  
**Cache:** `Cache-Control: public, max-age=3600`

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 100001, "name": "Genre", "slug": "genre", "sortorder": 1,
      "tags": [
        { "id": 100001, "name": "Action", "slug": "action", "description": null },
        { "id": 100002, "name": "Romance", "slug": "romance", "description": null }
      ]
    },
    { "id": 100002, "name": "Theme", "slug": "theme", "sortorder": 2, "tags": [...] },
    { "id": 100003, "name": "Format", "slug": "format", "sortorder": 3, "tags": [...] },
    { "id": 100004, "name": "Demographic", "slug": "demographic", "sortorder": 4, "tags": [...] }
  ]
}
```

---

### GET /tags/:id

Get a single tag by integer ID.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": {
    "id": 100001,
    "groupid": 100001,
    "group": { "id": 100001, "name": "Genre", "slug": "genre", "sortorder": 1 },
    "name": "Action",
    "slug": "action",
    "description": null
  }
}
```

---

### GET /tags/by-slug/:slug

Get a single tag by slug — useful for URL-based filter building.

**Auth required:** No

**Response `200 OK`:** Same as `GET /tags/:id`

---

## 6. Scanlation Groups

### GET /groups

List and search scanlation groups.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `name` |
| `isofficialpublisher` | bool | — | Filter official publishers only |
| `isfocused` | bool | — | Filter active groups only |
| `sort` | string | `name` | `name` \| `followcount` \| `createdat` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-...", "name": "MangaPlus (Official)", "slug": "mangaplus-official",
      "description": "Official digital service by Shueisha.",
      "website": "https://mangaplus.shueisha.co.jp",
      "isofficialpublisher": true, "isactive": true, "isfocused": true,
      "verifiedat": "2026-02-21T00:00:00Z",
      "membercount": 12, "followcount": 50321,
      "createdat": "2026-02-21T00:00:00Z", "updatedat": "2026-02-21T00:00:00Z"
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### GET /groups/:id

Full scanlation group detail.

**Auth required:** No  
**Path params:** `id` — UUIDv7

**Response `200 OK`:** Full `ScanlationGroup` object.

**Errors:** `404 NOT_FOUND`

---

### GET /groups/:id/chapters

Chapters uploaded by this group.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `language` | string | — | BCP-47 code filter |
| `sort` | string | `desc` | `asc` \| `desc` (by `publishedat`) |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa5-...", "comicid": "01952fa3-...",
      "comic": { "id": "01952fa3-...", "title": "One Piece", "coverurl": "..." },
      "chapternumber": 1110.0, "title": "The World",
      "language": { "code": "en", "name": "English" },
      "volume": 111, "pagecount": 17, "viewcount": 84231,
      "publishedat": "2026-02-21T00:00:00Z"
    }
  ],
  "meta": { "total": 200, "page": 1, "limit": 20, "pages": 10 }
}
```

---

### GET /groups/:id/members

List members of a group.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    {
      "userid": "01952fa3-...", "username": "buivan",
      "displayname": "Tài", "avatarurl": null,
      "role": "leader", "joinedat": "2026-02-21T00:00:00Z"
    }
  ]
}
```

---

### POST /groups

Create a new scanlation group.

**Auth required:** Yes (any verified `member`)

**Request body:**
```json
{
  "name": "Yomira Scans",
  "description": "Vietnamese scanlation group.",
  "website": "https://yomira.app",
  "discord": "https://discord.gg/abc123",
  "twitter": null, "patreon": null, "youtube": null, "mangaupdates": null
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `name` | string | Yes | Max 200 chars |
| `description` | string \| null | No | — |
| `website` | string \| null | No | Valid HTTPS URL |
| `discord` \| `twitter` \| `patreon` \| `youtube` \| `mangaupdates` | string \| null | No | Valid HTTPS URLs |

**Response `201 Created`:** Full `ScanlationGroup` object.

**Side effects:**
- `core.scanlationgroup` row created (`isofficialpublisher = FALSE`, `isactive = TRUE`)
- Caller auto-added as `leader` in `core.scanlationgroupmember`

---

### PATCH /groups/:id

Update group info.

**Auth required:** Yes (group `leader` or `moderator`, or `admin`)

**Request body:** All optional (partial update) — same fields as `POST /groups`.

> Group `leader`/`moderator` cannot change `isofficialpublisher`, `verifiedat`. Admin can change all.

**Response `200 OK`:** Updated `ScanlationGroup` object.

**Errors:**
```json
{ "error": "Only the group leader or moderator can update group info", "code": "FORBIDDEN" }
```

---

### POST /groups/:id/members

Add a member. Leader or admin only.

**Auth required:** Yes

**Request body:**
```json
{ "userid": "01952fa3-...", "role": "member" }
```

| Field | Validation |
|---|---|
| `userid` | Must be existing, active, verified user |
| `role` | `leader` \| `moderator` \| `member` |

**Response `201 Created`:** `ScanlationGroupMember` object.

**Side effects:** `core.scanlationgroupmember` row inserted.

**Errors:**
```json
{ "error": "User is already a member of this group", "code": "CONFLICT" }
{ "error": "Only group leaders can add members", "code": "FORBIDDEN" }
```

---

### PATCH /groups/:id/members/:userId/role

Change a member's role. Leader or admin only.

**Auth required:** Yes

**Request body:** `{ "role": "moderator" }`

**Response `200 OK`:** Updated `ScanlationGroupMember` object.

**Business rules:** A leader cannot demote themselves unless another leader exists.

---

### DELETE /groups/:id/members/:userId

Remove a member from the group.

**Auth required:** Yes

**Authorization:** Leader → can remove any member. Any member → can remove themselves. Admin → can remove anyone.

**Response `204 No Content`**

**Side effects:** `core.scanlationgroupmember` row hard-deleted.

**Errors:**
```json
{ "error": "Transfer leadership before leaving the group", "code": "FORBIDDEN" }
```

---

### POST /groups/:id/follow

Follow a group. New chapters appear in the user's Feed.

**Auth required:** Yes

**Response `201 Created`:**
```json
{ "data": { "groupid": "01952fa3-...", "createdat": "2026-02-21T23:00:00Z" } }
```

**Side effects:** `core.scanlationgroupfollow` row inserted.

**Errors:** `409 CONFLICT` if already following.

---

### DELETE /groups/:id/follow

Unfollow a group.

**Auth required:** Yes

**Response `204 No Content`**

**Side effects:** `core.scanlationgroupfollow` row hard-deleted.

---

### GET /me/groups

Groups the current user is a member of.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-...", "name": "Yomira Scans", "slug": "yomira-scans",
      "isofficialpublisher": false, "role": "leader",
      "joinedat": "2026-02-21T00:00:00Z"
    }
  ]
}
```

---

### GET /me/groups/following

Groups the current user follows.

**Auth required:** Yes  
**Query params:** `page`, `limit`

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "...", "name": "MangaPlus (Official)", "slug": "mangaplus-official",
      "isofficialpublisher": true, "followedat": "2026-02-21T00:00:00Z"
    }
  ],
  "meta": { "total": 3, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### PATCH /admin/groups/:id/verify

Grant or revoke the official publisher badge.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "isofficialpublisher": true, "reason": "Verified with Shueisha legal team" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `isofficialpublisher` | bool | Yes | `true` = grant badge, `false` = revoke |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:** Updated `ScanlationGroup` object.

**Side effects:** `core.scanlationgroup.isofficialpublisher` updated. If `true`: `verifiedat = NOW()`, `verifiedby = callerID`. If `false`: both cleared. `system.auditlog` written.

---

### PATCH /admin/groups/:id/suspend

Suspend or restore a scanlation group.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "suspend": true, "reason": "Group distributing pirated official content" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `suspend` | bool | Yes | `true` = suspend (`isactive = FALSE`), `false` = restore |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:** Updated `ScanlationGroup` object.

**Side effects:** `core.scanlationgroup.isactive` updated. `system.auditlog` written (action: `group.suspend` or `group.restore`).

---

### DELETE /admin/groups/:id

Soft-delete a scanlation group.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "reason": "Fraudulent group impersonating official publisher" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `reason` | string | No | Stored in `system.auditlog` |

**Response `204 No Content`**

**Side effects:** `core.scanlationgroup.deletedat = NOW()`. `system.auditlog` written.

---

## 7. Comics

### GET /comics

**The main discovery endpoint.** List, filter, and full-text search comics.

**Auth required:** No  
**Rate limit:** 60 requests/min

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Full-text search (title, alt titles, synopsis) |
| `status` | string | — | `ongoing` \| `completed` \| `hiatus` \| `cancelled` \| `unknown`. Multi-value allowed. |
| `contentrating` | string | `safe,suggestive` | Comma-separated. Auth users inherit `hidensfw` pref. |
| `demographic` | string | — | `shounen` \| `shoujo` \| `seinen` \| `josei`. Multi-value allowed. |
| `originlanguage` | string | — | BCP-47 code. Multi-value: `?originlanguage=ja&originlanguage=ko` |
| `includedtags` | int[] | — | Tag IDs comic **must have** (AND logic) |
| `excludedtags` | int[] | — | Tag IDs comic **must not have** |
| `includedauthors` | int[] | — | Author IDs (AND logic) |
| `includedartists` | int[] | — | Artist IDs (AND logic) |
| `availablelanguage` | string | — | Show comics with ≥1 chapter in this language |
| `year` | int | — | Publication year |
| `sort` | string | `latest` | `latest` (publishedat) \| `popular` (viewcount) \| `rating` (ratingbayesian) \| `followcount` \| `az` \| `za` \| `createdat` |
| `page` | int | `1` | — |
| `limit` | int | `24` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-3f1e-7abc-b12e-1234567890ab",
      "title": "One Piece", "slug": "one-piece",
      "coverurl": "https://cdn.yomira.app/covers/onepiece.webp",
      "status": "ongoing", "contentrating": "safe", "demographic": "shounen",
      "originlanguage": "ja", "year": 1997,
      "ratingbayesian": 9.12, "followcount": 984321, "chaptercount": 1110,
      "tags": [
        { "id": 100001, "name": "Action", "slug": "action" },
        { "id": 100002, "name": "Adventure", "slug": "adventure" }
      ],
      "latestchapter": {
        "id": "01952fa5-...", "chapternumber": 1110.0,
        "publishedat": "2026-02-21T00:00:00Z",
        "language": { "code": "en", "name": "English" }
      }
    }
  ],
  "meta": { "total": 52000, "page": 1, "limit": 24, "pages": 2167 }
}
```

---

### GET /comics/:id

Full comic detail. Accepts UUIDv7 **or** slug.

**Auth required:** No (authenticated users get `isFollowing`, `userRating`, `readingStatus` fields)

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01952fa3-...", "title": "One Piece", "slug": "one-piece",
    "synopsis": "Gol D. Roger was known as the Pirate King...",
    "coverurl": "https://cdn.yomira.app/covers/onepiece.webp", "bannerurl": null,
    "status": "ongoing", "contentrating": "safe", "demographic": "shounen",
    "defaultreadmode": "ltr", "originlanguage": "ja", "year": 1997,
    "links": { "mal": "13", "anilist": "100526", "official": "https://mangaplus.shueisha.co.jp" },
    "viewcount": 102843921, "followcount": 984321, "chaptercount": 1110,
    "ratingavg": 9.18, "ratingbayesian": 9.12, "ratingcount": 58421,
    "islocked": false,
    "authors": [{ "id": 100001, "name": "Eiichiro Oda", "namealt": ["尾田 栄一郎"] }],
    "artists": [{ "id": 100001, "name": "Eiichiro Oda", "namealt": ["尾田 栄一郎"] }],
    "tags": [{ "id": 100001, "name": "Action", "slug": "action", "group": { "name": "Genre" } }],
    "relations": [
      { "relatedcomic": { "id": "...", "title": "One Piece Film Gold" }, "relationtype": "side_story" }
    ],
    "covers": [{ "id": "...", "volume": 1, "imageurl": "...", "description": null }],
    "createdat": "2026-02-21T00:00:00Z", "updatedat": "2026-02-21T00:00:00Z",
    "isFollowing": false, "userRating": null,
    "readingStatus": null, "lastReadChapterNumber": null
  }
}
```

**Errors:** `404 NOT_FOUND`

---

### GET /comics/:id/chapters

Paginated chapter list for a comic.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `language` | string | — | BCP-47. Multi: `?language=en&language=vi` |
| `group` | string | — | Scanlation group UUIDv7 |
| `volume` | int | — | Filter by volume |
| `sort` | string | `desc` | `asc` \| `desc` by `chapternumber`, then `publishedat` |
| `page` | int | `1` | — |
| `limit` | int | `96` | Max `500` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa5-...", "comicid": "01952fa3-...",
      "volume": 111, "chapternumber": 1110.0, "title": "The World",
      "language": { "code": "en", "name": "English" },
      "scanlationgroup": { "id": "...", "name": "MangaPlus (Official)", "isofficialpublisher": true },
      "isofficial": true, "islocked": false,
      "pagecount": 17, "viewcount": 84231,
      "publishedat": "2026-02-21T00:00:00Z",
      "isRead": false
    }
  ],
  "meta": { "total": 1110, "page": 1, "limit": 96, "pages": 12 }
}
```

> `isRead` only populated when caller is authenticated (from `library.chapterread`).

---

### POST /comics

Create a new comic.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{
  "title": "One Piece",
  "titlealt": ["ワンピース", "Vua Hải Tặc"],
  "synopsis": "Gol D. Roger was known as the Pirate King...",
  "status": "ongoing", "contentrating": "safe", "demographic": "shounen",
  "defaultreadmode": "ltr", "originlanguage": "ja", "year": 1997,
  "links": { "mal": "13", "anilist": "100526" },
  "authorids": [100001], "artistids": [100001], "tagids": [100001, 100002]
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `title` | string | Yes | Max 500 chars |
| `titlealt` | string[] | No | Each max 500 chars |
| `synopsis` | string \| null | No | — |
| `status` | string | Yes | One of 5 allowed values |
| `contentrating` | string | Yes | `safe` \| `suggestive` \| `explicit` |
| `demographic` | string \| null | No | `shounen` \| `shoujo` \| `seinen` \| `josei` |
| `defaultreadmode` | string | No | `ltr` \| `rtl` \| `vertical` \| `webtoon` (default: `ltr`) |
| `originlanguage` | string \| null | No | Valid BCP-47 code |
| `year` | int \| null | No | 1900–(currentYear + 1) |
| `links` | object | No | Keys: `mal`, `anilist`, `official`, `raw`, `kitsu`, `mangaupdates` |
| `authorids` | int[] | No | Must exist in `core.author` |
| `artistids` | int[] | No | Must exist in `core.artist` |
| `tagids` | int[] | No | Must exist in `core.tag` |

**Response `201 Created`:** Full `ComicDetail` object.

**Side effects:** `core.comic` + junction rows created. Slug auto-generated. `system.auditlog` written.

---

### PATCH /comics/:id

Update comic metadata. Admin or moderator.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:** All fields optional (partial update) — same shape as `POST /comics`.

> `authorids`, `artistids`, `tagids` use **full replacement**: send complete new array.

**Response `200 OK`:** Updated `ComicDetail` object.

**Side effects:** Junction rows re-synced. `searchvector` trigger fires. `system.auditlog` written.

---

### DELETE /comics/:id

Soft-delete a comic.

**Auth required:** Yes (role: `admin`)

**Response `204 No Content`**

**Side effects:** `core.comic.deletedat = NOW()`. `system.auditlog` written.

---

### PATCH /admin/comics/:id/lock

Toggle moderation lock — prevents community metadata edits.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "islocked": true, "reason": "Official publisher requested" }
```

**Response `200 OK`:** `{ "id": "...", "islocked": true, "updatedat": "..." }`

---

## 8. Comic Titles

### GET /comics/:id/titles

List all per-language titles (multilingual support).

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    { "languageid": 100001, "language": { "code": "en", "name": "English" }, "title": "One Piece", "description": "Gol D. Roger..." },
    { "languageid": 100003, "language": { "code": "ja", "name": "Japanese" }, "title": "ワンピース", "description": null },
    { "languageid": 100002, "language": { "code": "vi", "name": "Vietnamese" }, "title": "Vua Hải Tặc", "description": "..." }
  ]
}
```

---

### PUT /comics/:id/titles/:languageCode

Upsert title + description for a specific language.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{ "title": "Vua Hải Tặc", "description": "Gol D. Roger được biết đến là Vua Hải Tặc..." }
```

| Field | Required | Validation |
|---|---|---|
| `title` | Yes | Max 500 chars |
| `description` | No | — |

**Response `200 OK`:**
```json
{
  "data": {
    "languageid": 100002, "language": { "code": "vi", "name": "Vietnamese" },
    "title": "Vua Hải Tặc", "description": "...", "updatedat": "2026-02-21T23:43:33Z"
  }
}
```

**Side effects:** `INSERT ... ON CONFLICT (comicid, languageid) DO UPDATE SET ...`

---

### DELETE /comics/:id/titles/:languageCode

Remove title entry for a language.

**Auth required:** Yes (role: `admin` | `moderator`)

**Response `204 No Content`**

---

## 9. Comic Relations

### GET /comics/:id/relations

Get all directional relations for a comic.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    { "direction": "from", "relatedcomic": { "id": "...", "title": "One Piece Film: Gold", "coverurl": "..." }, "relationtype": "side_story" },
    { "direction": "to",   "relatedcomic": { "id": "...", "title": "Romance Dawn", "coverurl": "..." }, "relationtype": "preserialization" }
  ]
}
```

---

### POST /comics/:id/relations

Add a relation from this comic to another.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{ "tocomicid": "01952fb0-...", "relationtype": "side_story" }
```

| Field | Required | Validation |
|---|---|---|
| `tocomicid` | Yes | Must exist; cannot equal `:id` |
| `relationtype` | Yes | One of 10 allowed values |

**Response `201 Created`:** Created relation object.

**Errors:**
```json
{ "error": "This relation already exists", "code": "CONFLICT" }
{ "error": "Cannot create a relation to itself", "code": "VALIDATION_ERROR" }
```

---

### DELETE /comics/:id/relations/:toComicId/:relationType

Remove a specific relation.

**Auth required:** Yes (role: `admin` | `moderator`)

**Response `204 No Content`**

---

## 10. Comic Covers

### GET /comics/:id/covers

All volume covers, ordered by volume number.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    { "id": "...", "volume": null, "imageurl": "...", "description": "Default cover", "createdat": "..." },
    { "id": "...", "volume": 1,    "imageurl": "...", "description": null, "createdat": "..." }
  ]
}
```

> `volume: null` = default series cover.

---

### POST /comics/:id/covers

Upload a volume cover.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Content-Type:** `multipart/form-data`

**Request body:**
```
image: <file>         (JPEG/PNG/WEBP, max 10 MB)
volume: 1             (optional integer)
description: "..."    (optional)
```

**Response `201 Created`:** New `ComicCover` object.

**Side effects:** Image → object storage → `core.mediafile` row + `core.comiccover` row. If `volume = null`: `core.comic.coverurl` updated.

---

### DELETE /comics/:id/covers/:coverId

Delete a cover image.

**Auth required:** Yes (role: `admin` | `moderator`)

**Response `204 No Content`**

---

## 11. Comic Art Gallery

### GET /comics/:id/art

**Auth required:** No

**Query params:**

| Param | Type | Description |
|---|---|---|
| `type` | string | `cover` \| `banner` \| `fanart` \| `promotional` \| `chapter_cover` |
| `page` | int | Default `1` |
| `limit` | int | Default `20`, max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "...", "comicid": "...", "arttype": "promotional",
      "volume": null, "imageurl": "https://cdn.yomira.app/art/...",
      "width": 1920, "height": 1080,
      "description": "Chapter 1000 Celebration Art",
      "isapproved": true,
      "uploader": { "id": "...", "username": "buivan", "avatarurl": null },
      "createdat": "2026-02-21T00:00:00Z"
    }
  ],
  "meta": { "total": 45, "page": 1, "limit": 20, "pages": 3 }
}
```

---

### POST /comics/:id/art

Upload an art image.

**Auth required:** Yes (any verified member for `fanart`; admin/mod for all types)  
**Content-Type:** `multipart/form-data`

**Request body:**
```
image: <file>         (JPEG/PNG/WEBP/GIF, max 10 MB)
arttype: "fanart"
volume: 1             (optional)
description: "..."    (optional)
```

**Response `201 Created`:** New art object.

**Side effects:** `isapproved = TRUE` for admin/mod; `FALSE` for member fanart (pending review).

---

### PATCH /admin/comics/:id/art/:artId/approve

Approve or reject pending fanart.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:** `{ "isapproved": true }`

**Response `200 OK`:** Updated art object.

---

### DELETE /comics/:id/art/:artId

Delete an art image (uploader or admin/mod).

**Auth required:** Yes

**Business rule:** User can delete their own uploads. Admin/mod can delete any.

**Response `204 No Content`**

---

## 12. Chapters

### GET /chapters/:id

Chapter detail **including full page list** — the primary reader endpoint.

**Auth required:** No  
**Path params:** `id` — UUIDv7

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01952fa5-...", "comicid": "01952fa3-...",
    "volume": 111, "chapternumber": 1110.0, "title": "The World",
    "language": { "code": "en", "name": "English" },
    "scanlationgroup": { "id": "...", "name": "MangaPlus (Official)", "isofficialpublisher": true },
    "isofficial": true, "islocked": false,
    "externalurl": "https://mangaplus.shueisha.co.jp/viewer/...",
    "pagecount": 17, "viewcount": 84231,
    "publishedat": "2026-02-21T00:00:00Z",
    "createdat": "2026-02-21T00:00:00Z", "updatedat": "2026-02-21T00:00:00Z",
    "pages": [
      {
        "id": "...", "pagenumber": 1,
        "imageurl": "https://cdn.yomira.app/pages/ch1110/p001.webp",
        "imageurlhd": "https://cdn.yomira.app/pages/ch1110/hd/p001.webp",
        "format": "webp", "width": 784, "height": 1200, "filesizebytes": 204800
      }
    ],
    "prevchapter": { "id": "...", "chapternumber": 1109.0, "language": { "code": "en" } },
    "nextchapter": null,
    "isRead": false
  }
}
```

> **Data Saver mode:** If `users.readingpreference.datasaver = TRUE`, return compressed `imageurl` and omit `imageurlhd`.

**Side effects (async, fire-and-forget):**
- `analytics.pageview` inserted (type: `chapter_view`)
- Redis `INCR chapter:{id}:views` + `INCR comic:{comicid}:views`

**Errors:** `404 NOT_FOUND`

---

### POST /chapters

Upload a new chapter. Caller must be a member of the specified group (or admin/mod).

**Auth required:** Yes

**Request body:**
```json
{
  "comicid": "01952fa3-...",
  "languageid": 100001,
  "scanlationgroupid": "01952fa4-...",
  "volume": 111,
  "chapternumber": 1110.0,
  "title": "The World",
  "publishedat": "2026-02-21T00:00:00Z",
  "externalurl": null,
  "isofficial": false
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `comicid` | string | Yes | Must exist, not deleted |
| `languageid` | int | Yes | Must exist in `core.language` |
| `scanlationgroupid` | string | Yes | Must exist. Caller must be a member (or admin/mod). |
| `volume` | int \| null | No | — |
| `chapternumber` | number | Yes | ≥ 0. NUMERIC(8,2): supports 12.5 |
| `title` | string \| null | No | Max 500 chars |
| `publishedat` | string \| null | No | ISO 8601. Default: `NOW()` |
| `externalurl` | string \| null | No | Valid HTTPS URL |
| `isofficial` | bool | No | Only admin/mod can set `true` |

**Response `201 Created`:** Full `Chapter` object (without pages — upload pages separately via `POST /chapters/:id/pages`).

**Side effects:**
- `core.chapter` row created (`syncstate = 'pending'`)
- `core.comic.chaptercount` incremented
- `library.entry.hasnew = TRUE` for all followers (async background job)
- `social.feedevent` created (type: `chapter_uploaded`)

**Errors:**
```json
{ "error": "You are not a member of the specified scanlation group", "code": "FORBIDDEN" }
{ "error": "Chapter already exists for this comic, language, and group", "code": "CONFLICT" }
```

---

### POST /chapters/:id/pages

Upload pages for a chapter in batch. Called after `POST /chapters`.

**Auth required:** Yes (same group member or admin/mod)  
**Content-Type:** `multipart/form-data`

**Request body:**
```
pages[0]: <file>     (pagenumber auto-assigned from array index: 0→page 1)
pages[1]: <file>
...
```

**Constraints:** Max 500 pages/chapter. Formats: JPEG, PNG, WEBP, GIF. Max 10 MB each.

**Response `202 Accepted`:**
```json
{
  "data": {
    "chapterid": "01952fa5-...", "uploadjobid": "job_abc123",
    "pagecount": 17, "status": "processing"
  }
}
```

**Side effects:**
- Files uploaded to object storage; WebP conversion + HD extract queued async
- `core.chapter.syncstate = 'processing'`
- On job complete: `core.page` rows created, `core.chapter.pagecount` updated, `syncstate = 'synced'`

---

### PATCH /chapters/:id

Update chapter metadata.

**Auth required:** Yes (group leader/mod or admin/mod)

**Request body:** All optional (partial update).
```json
{ "volume": 112, "chapternumber": 1110.5, "title": "Updated Title", "publishedat": "2026-02-22T00:00:00Z" }
```

> `comicid`, `languageid`, `scanlationgroupid` are **immutable** after creation.

**Response `200 OK`:** Updated `Chapter` object (without pages).

---

### DELETE /chapters/:id

Soft-delete a chapter.

**Auth required:** Yes (group leader, admin, or moderator)

**Response `204 No Content`**

**Side effects:** `core.chapter.deletedat = NOW()`. `core.comic.chaptercount` decremented. `system.auditlog` written.

---

### POST /chapters/:id/read

Mark a chapter as fully read.

**Auth required:** Yes

**Request body:** None

**Response `204 No Content`**

**Side effects:**
- `library.chapterread` inserted (`ON CONFLICT DO NOTHING`)
- `library.readingprogress` upserted for `(userid, comicid)`
- `library.entry.lastreadchapterid` + `lastreadat` updated
- `library.entry.hasnew` recalculated

---

### PATCH /admin/chapters/:id/lock

Lock or unlock a chapter. A locked chapter cannot be modified by group members.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:**
```json
{ "islocked": true, "reason": "Official publisher DMCA request" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `islocked` | bool | Yes | `true` = lock, `false` = unlock |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:**
```json
{ "data": { "id": "01952fa5-...", "islocked": true, "updatedat": "2026-02-22T00:02:08Z" } }
```

**Side effects:** `core.chapter.islocked` updated. `system.auditlog` written.

---

### PATCH /admin/chapters/:id/official

Mark or unmark a chapter as an official release (e.g. publisher-sponsored translation).

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{ "isofficial": true, "reason": "MangaPlus official EN release" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `isofficial` | bool | Yes | `true` = mark official, `false` = remove official flag |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:**
```json
{ "data": { "id": "01952fa5-...", "isofficial": true, "updatedat": "2026-02-22T00:02:08Z" } }
```

**Side effects:** `core.chapter.isofficial` updated. `system.auditlog` written.

---

### GET /admin/chapters

List chapters by sync state — used to monitor the page upload pipeline.

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `syncstate` | string | — | `pending` \| `processing` \| `synced` \| `failed` \| `missing` |
| `comicid` | string | — | Filter by comic |
| `since` | string | — | ISO 8601 datetime — chapters created after this time |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa5-...",
      "comicid": "01952fa3-...",
      "comic": { "id": "...", "title": "One Piece" },
      "chapternumber": 1110.0,
      "language": { "code": "en" },
      "scanlationgroup": { "id": "...", "name": "MangaPlus (Official)" },
      "syncstate": "failed",
      "pagecount": 0,
      "createdat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 7, "page": 1, "limit": 50, "pages": 1 }
}
```

---

## 13. Pages

Pages are **bundled in `GET /chapters/:id`** for the reader. Standalone endpoints for special cases only.

### GET /chapters/:id/pages

Standalone page list — for clients that want metadata separately.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "...", "chapterid": "01952fa5-...", "pagenumber": 1,
      "imageurl": "https://cdn.yomira.app/pages/ch1110/p001.webp",
      "imageurlhd": "https://cdn.yomira.app/pages/ch1110/hd/p001.webp",
      "format": "webp", "width": 784, "height": 1200, "filesizebytes": 204800
    }
  ]
}
```

> Returns `imageurl` vs `imageurlhd` based on caller's `datasaver` preference.

---

### DELETE /admin/chapters/:id/pages/:pageNumber

Delete a specific page (rare — DMCA or content violation).

**Auth required:** Yes (role: `admin`)

**Response `204 No Content`**

**Side effects:** `core.page` hard-deleted. `core.chapter.pagecount` decremented. Object storage cleanup queued.

---

## 14. Implementation Notes

### Go validation (NOT SQL)

```go
// core.comic
status:          "ongoing" | "completed" | "hiatus" | "cancelled" | "unknown"
contentrating:   "safe" | "suggestive" | "explicit"
demographic:     "shounen" | "shoujo" | "seinen" | "josei" | nil
defaultreadmode: "ltr" | "rtl" | "vertical" | "webtoon"
year:            1900 <= year <= currentYear+1 (if not nil)
originlanguage:  must exist in core.language.code (if not nil)

// core.chapter
chapternumber:      >= 0 (NUMERIC 8,2 → 0.00 to 999999.99)
scanlationgroupid:  caller must be group member OR role admin|moderator
isofficial:         only admin|moderator can set true

// core.comicrelation
fromcomicid != tocomicid  (self-relation prevention)
relationtype: "sequel"|"prequel"|"main_story"|"side_story"|"spin_off"|
              "adaptation"|"alternate"|"same_franchise"|"colored"|"preserialization"

// core.scanlationgroupmember
role: "leader" | "moderator" | "member"
```

### Key SQL patterns

**Included tags (AND logic):**
```sql
AND NOT EXISTS (
    SELECT 1 FROM UNNEST($tagids::int[]) AS t(id)
    WHERE t.id NOT IN (
        SELECT tagid FROM core.comictag WHERE comicid = c.id
    )
)
```

**Content rating guard:**
```sql
-- hidensfw = TRUE (or unauthenticated):
AND c.contentrating IN ('safe', 'suggestive')
-- hidensfw = FALSE (authenticated, explicit opted in):
AND c.contentrating IN ('safe', 'suggestive', 'explicit')
```

**ID or slug resolution:**
```go
func resolveComicID(ctx context.Context, idOrSlug string) (string, error) {
    if isUUID(idOrSlug) { return idOrSlug, nil }
    return comicRepo.FindIDBySlug(ctx, idOrSlug)
}
```

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /languages` | 24h | `lang:all` |
| `GET /tags` | 1h | `tags:all` |
| `GET /groups/:id` | 5 min | `group:{id}` |
| `GET /authors/:id` | 10 min | `author:{id}` |
| `GET /comics/:id` | 2 min | `comic:{id}` |
| `GET /comics/:id/chapters` | 1 min | `comic:{id}:ch:p{n}:l{lang}` |
| `GET /chapters/:id` (pages) | 10 min | `chapter:{id}` |
| `GET /comics` list | 30 sec | `comics:list:{sort}:p{n}` |

> Cache invalidated on any write affecting the resource.
