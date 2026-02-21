# API Reference — Social Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `40_SOCIAL/SOCIAL.sql`

> Global conventions (auth header, response envelope, error codes, pagination) — see [USERS_API.md](./USERS_API.md#global-conventions).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Ratings, Comments, Notifications, Recommendations, Feed, Forum, Reports. |

---

## Table of Contents

1. [Common Types](#1-common-types)
2. [Comic Ratings](#2-comic-ratings)
3. [Comments](#3-comments)
4. [Notifications](#4-notifications)
5. [Recommendations](#5-recommendations)
6. [Activity Feed](#6-activity-feed)
7. [Forum](#7-forum)
8. [Reports](#8-reports)
9. [Implementation Notes](#9-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/comics/:id/rating` | No | Get aggregate rating stats for a comic |
| `PUT` | `/comics/:id/rating` | Yes | Submit / update personal rating (1–10) |
| `DELETE` | `/comics/:id/rating` | Yes | Remove personal rating |
| `GET` | `/comics/:id/comments` | No | List top-level comments on a comic |
| `GET` | `/chapters/:id/comments` | No | List top-level comments on a chapter |
| `GET` | `/comments/:id/replies` | No | List replies to a comment |
| `POST` | `/comics/:id/comments` | Yes | Post a comment on a comic |
| `POST` | `/chapters/:id/comments` | Yes | Post a comment on a chapter |
| `POST` | `/comments/:id/replies` | Yes | Reply to a comment |
| `PATCH` | `/comments/:id` | Yes | Edit own comment |
| `DELETE` | `/comments/:id` | Yes | Soft-delete own comment |
| `POST` | `/comments/:id/vote` | Yes | Upvote or downvote a comment |
| `DELETE` | `/comments/:id/vote` | Yes | Remove vote from a comment |
| `GET` | `/me/notifications` | Yes | List user notifications |
| `GET` | `/me/notifications/unread-count` | Yes | Get unread notification count |
| `PATCH` | `/me/notifications/:id/read` | Yes | Mark a notification as read |
| `PATCH` | `/me/notifications/read-all` | Yes | Mark all notifications as read |
| `DELETE` | `/me/notifications/:id` | Yes | Delete a notification |
| `GET` | `/comics/:id/recommendations` | No | List community recommendations for a comic |
| `POST` | `/comics/:id/recommendations` | Yes | Submit a recommendation |
| `POST` | `/recommendations/:id/vote` | Yes | Upvote or downvote a recommendation |
| `DELETE` | `/recommendations/:id/vote` | Yes | Remove recommendation vote |
| `DELETE` | `/recommendations/:id` | Yes | Delete own recommendation |
| `GET` | `/me/feed` | Yes | Get personalized activity feed |
| `GET` | `/forums` | No | List all forum boards |
| `GET` | `/forums/:slug` | No | Get a forum board detail |
| `GET` | `/forums/:slug/threads` | No | List threads in a forum board |
| `POST` | `/forums/:slug/threads` | Yes | Create a new thread |
| `GET` | `/threads/:id` | No | Get thread detail |
| `GET` | `/threads/:id/posts` | No | List posts in a thread |
| `POST` | `/threads/:id/posts` | Yes | Reply to a thread |
| `PATCH` | `/posts/:id` | Yes | Edit own forum post |
| `DELETE` | `/posts/:id` | Yes | Soft-delete own forum post |
| `POST` | `/posts/:id/vote` | Yes | Vote on a forum post |
| `DELETE` | `/posts/:id/vote` | Yes | Remove vote from a forum post |
| `PATCH` | `/admin/threads/:id/pin` | admin/mod | Pin or unpin a thread |
| `PATCH` | `/admin/threads/:id/lock` | admin/mod | Lock or unlock a thread |
| `DELETE` | `/admin/threads/:id` | admin/mod | Soft-delete a thread |
| `PATCH` | `/admin/forums/:slug/archive` | admin | Archive or restore a forum board |
| `POST` | `/reports` | Yes | Submit a content report |
| `GET` | `/admin/reports` | admin/mod | List moderation reports |
| `PATCH` | `/admin/reports/:id` | admin/mod | Update report status / resolve |

---

## 1. Common Types

### `ComicRating`
```typescript
{
  ratingavg: number          // arithmetic average
  ratingbayesian: number     // Bayesian weighted average (displayed to users)
  ratingcount: number
  userRating: number | null  // caller's own score (null if not rated / unauthenticated)
}
```

### `Comment`
```typescript
{
  id: string                 // UUIDv7
  userid: string
  author: { id: string; username: string; avatarurl: string | null }
  comicid: string | null
  chapterid: string | null
  parentid: string | null
  body: string               // raw text (rendered client-side)
  isedited: boolean
  isdeleted: boolean         // if true, body replaced with "[deleted]" in response
  isapproved: boolean
  upvotes: number
  downvotes: number
  replycount: number         // denormalized
  userVote: 1 | -1 | null   // caller's current vote (null if not voted / unauth)
  createdat: string
  updatedat: string
}
```

### `Notification`
```typescript
{
  id: string
  userid: string
  type: "new_chapter" | "system" | "announcement" | "comment_reply" | "follow"
  title: string
  body: string | null
  entitytype: "comic" | "chapter" | "comment" | "user" | null
  entityid: string | null    // for deep-link navigation
  isread: boolean
  createdat: string
}
```

### `ComicRecommendation`
```typescript
{
  id: number
  fromcomicid: string
  tocomic: ComicSummary      // the recommended comic
  submittedby: { id: string; username: string; avatarurl: string | null }
  reason: string | null
  upvotes: number
  userVote: 1 | -1 | null
  createdat: string
}
```

### `FeedEvent`
```typescript
{
  id: string
  eventtype: "chapter_published" | "comic_followed" | "user_followed" |
             "group_chapter" | "list_updated" | "comic_updated"
  actor: { id: string; username: string; avatarurl: string | null } | null
  entitytype: string
  entityid: string
  payload: object            // shape varies by eventtype — see §6
  createdat: string
}
```

### `Forum`
```typescript
{
  id: number
  comicid: string | null     // null = site-wide board
  name: string; slug: string; description: string | null
  sortorder: number
  isarchived: boolean
  canpost: "admin" | "moderator" | "member"
  threadcount: number
  postcount: number
}
```

### `ForumThread`
```typescript
{
  id: string
  forumid: number
  author: { id: string; username: string; avatarurl: string | null }
  title: string
  ispinned: boolean; islocked: boolean; isdeleted: boolean
  replycount: number; viewcount: number
  lastpostedat: string | null
  lastposter: { id: string; username: string } | null
  createdat: string
}
```

### `ForumPost`
```typescript
{
  id: string
  threadid: string
  author: { id: string; username: string; avatarurl: string | null }
  body: string
  bodyformat: "markdown" | "plain"
  isedited: boolean; isdeleted: boolean; isapproved: boolean
  upvotes: number; downvotes: number
  userVote: 1 | -1 | null
  createdat: string; updatedat: string
}
```

### `Report`
```typescript
{
  id: string
  reporter: { id: string; username: string }
  entitytype: "comic" | "chapter" | "comment" | "forumpost" | "user" | "scanlationgroup"
  entityid: string
  reason: "spam" | "violence" | "explicit_content" | "misinformation" |
          "copyright" | "duplicate" | "low_quality" | "other"
  details: string | null
  status: "open" | "reviewing" | "resolved" | "dismissed"
  resolvedby: string | null
  resolvedat: string | null
  resolution: string | null
  createdat: string
}
```

---

## 2. Comic Ratings

### GET /comics/:id/rating

Get aggregate rating stats for a comic.

**Auth required:** No (authenticated users also receive their own `userRating`)  
**Path params:** `id` — comic UUIDv7

**Response `200 OK`:**
```json
{
  "data": {
    "ratingavg": 9.18,
    "ratingbayesian": 9.12,
    "ratingcount": 58421,
    "userRating": 10
  }
}
```

---

### PUT /comics/:id/rating

Submit or update the current user's rating for a comic.

**Auth required:** Yes  
**Path params:** `id` — comic UUIDv7

**Request body:**
```json
{ "score": 10 }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `score` | int | Yes | 1–10 (Go-validated). Integer only. |

**Response `200 OK`:**
```json
{
  "data": {
    "score": 10,
    "ratingavg": 9.19,
    "ratingbayesian": 9.13,
    "ratingcount": 58422
  }
}
```

**Side effects:**
- `social.comicrating` upserted (`ON CONFLICT (userid, comicid) DO UPDATE SET score`)
- `core.comic.ratingavg` + `ratingbayesian` recalculated (background job or immediate trigger)

**Errors:**
```json
{ "error": "Score must be between 1 and 10", "code": "VALIDATION_ERROR" }
{ "error": "Comic not found", "code": "NOT_FOUND" }
```

---

### DELETE /comics/:id/rating

Remove the current user's rating.

**Auth required:** Yes  
**Path params:** `id` — comic UUIDv7

**Response `204 No Content`**

**Side effects:** `social.comicrating` row hard-deleted. `core.comic` aggregates recalculated.

**Errors:**
```json
{ "error": "You have not rated this comic", "code": "NOT_FOUND" }
```

---

## 3. Comments

Comments are attached to **either** a comic **or** a chapter — never both. The Go service enforces this XOR constraint. Soft-deleted comments (`isdeleted = TRUE`) remain in the thread; their `body` is replaced with `"[deleted]"` in the response.

### GET /comics/:id/comments

List top-level comments on a comic (paginated, newest first).

**Auth required:** No  
**Path params:** `id` — comic UUIDv7

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `sort` | string | `new` | `new` (createdat DESC) \| `top` (upvotes DESC) |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fd0-...",
      "author": { "id": "01952fa3-...", "username": "buivan", "avatarurl": null },
      "comicid": "01952fb0-...", "chapterid": null, "parentid": null,
      "body": "Best comic ever!",
      "isedited": false, "isdeleted": false, "isapproved": true,
      "upvotes": 42, "downvotes": 1, "replycount": 5,
      "userVote": null,
      "createdat": "2026-02-22T00:00:00Z", "updatedat": "2026-02-22T00:00:00Z"
    }
  ],
  "meta": { "total": 320, "page": 1, "limit": 20, "pages": 16 }
}
```

---

### GET /chapters/:id/comments

List top-level comments on a chapter.

**Auth required:** No  
**Path params:** `id` — chapter UUIDv7  
**Query params:** same as `GET /comics/:id/comments`

**Response `200 OK`:** Same shape as comic comments.

---

### GET /comments/:id/replies

List replies to a specific comment.

**Auth required:** No  
**Path params:** `id` — comment UUIDv7  
**Query params:** `page`, `limit` (default 20, max 100)

**Response `200 OK`:** Same `Comment` shape, all with `parentid = :id`.

---

### POST /comics/:id/comments

Post a top-level comment on a comic.

**Auth required:** Yes (verified member)

**Request body:**
```json
{ "body": "Best comic ever!" }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `body` | string | Yes | 1–10 000 chars. Stripped of leading/trailing whitespace. |

**Response `201 Created`:** New `Comment` object.

**Side effects:**
- `social.comment` row created (`comicid = :id`, `chapterid = NULL`, `parentid = NULL`)
- `isapproved = FALSE` for new accounts (< 5 total comments) — held for moderation

**Errors:**
```json
{ "error": "Body must be between 1 and 10000 characters", "code": "VALIDATION_ERROR" }
{ "error": "Comic not found", "code": "NOT_FOUND" }
```

---

### POST /chapters/:id/comments

Post a top-level comment on a chapter.

**Auth required:** Yes (verified member)  
**Request body:** Same as `POST /comics/:id/comments`.

**Response `201 Created`:** New `Comment` object (`chapterid = :id`, `comicid = NULL`).

---

### POST /comments/:id/replies

Reply to an existing comment.

**Auth required:** Yes (verified member)  
**Path params:** `id` — parent comment UUIDv7

**Request body:**
```json
{ "body": "I totally agree!" }
```

**Response `201 Created`:** New `Comment` object with `parentid = :id`.

**Business rules:**
- Replies are **flat** — replying to a reply still sets `parentid` to the **root** comment (depth limited to 1 level in UI, but stored flat).
- Parent comment must exist and not be deleted.

**Errors:**
```json
{ "error": "Cannot reply to a deleted comment", "code": "VALIDATION_ERROR" }
{ "error": "Comment not found", "code": "NOT_FOUND" }
```

---

### PATCH /comments/:id

Edit the body of an existing comment.

**Auth required:** Yes (comment author only)  
**Path params:** `id` — comment UUIDv7

**Request body:**
```json
{ "body": "Updated comment body." }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `body` | string | Yes | 1–10 000 chars |

**Response `200 OK`:** Updated `Comment` object (`isedited = true`).

**Side effects:** `social.comment.body` updated, `isedited = TRUE`.

**Errors:**
```json
{ "error": "Comment not found or you do not own this comment", "code": "FORBIDDEN" }
{ "error": "Cannot edit a deleted comment", "code": "VALIDATION_ERROR" }
```

---

### DELETE /comments/:id

Soft-delete a comment. The row is preserved for thread continuity.

**Auth required:** Yes (author, or admin/mod)

**Response `204 No Content`**

**Side effects:** `social.comment.isdeleted = TRUE`. Body replaced with `"[deleted]"` in API responses. Child replies are preserved.

---

### POST /comments/:id/vote

Upvote or downvote a comment.

**Auth required:** Yes  
**Path params:** `id` — comment UUIDv7

**Request body:**
```json
{ "vote": 1 }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `vote` | int | Yes | `1` (upvote) or `-1` (downvote) |

**Response `200 OK`:**
```json
{ "data": { "upvotes": 43, "downvotes": 1, "userVote": 1 } }
```

**Side effects:**
- `social.commentvote` upserted (`ON CONFLICT (userid, commentid) DO UPDATE SET vote`)
- `social.comment.upvotes` / `downvotes` updated (delta applied)
- Cannot vote on own comment (Go enforces)

**Errors:**
```json
{ "error": "Vote must be 1 or -1", "code": "VALIDATION_ERROR" }
{ "error": "Cannot vote on your own comment", "code": "FORBIDDEN" }
```

---

### DELETE /comments/:id/vote

Remove the current user's vote from a comment.

**Auth required:** Yes  
**Path params:** `id` — comment UUIDv7

**Response `200 OK`:**
```json
{ "data": { "upvotes": 42, "downvotes": 1, "userVote": null } }
```

**Side effects:** `social.commentvote` row deleted. `upvotes`/`downvotes` counter corrected.

**Errors:**
```json
{ "error": "You have not voted on this comment", "code": "NOT_FOUND" }
```

---

## 4. Notifications

Pull-based notification inbox. The server writes notifications on key events; the client polls.

### GET /me/notifications

List notifications for the current user.

**Auth required:** Yes

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `isread` | bool | — | `false` = unread only; `true` = read only; omit = all |
| `type` | string | — | Filter by type: `new_chapter` \| `comment_reply` \| `follow` \| `system` \| `announcement` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fe0-...",
      "type": "new_chapter",
      "title": "Solo Leveling — Chapter 180 is out!",
      "body": "MangaPlus (Official) has uploaded Chapter 180.",
      "entitytype": "chapter",
      "entityid": "01952fa5-...",
      "isread": false,
      "createdat": "2026-02-22T00:12:40Z"
    },
    {
      "id": "01952fe1-...",
      "type": "comment_reply",
      "title": "buivan replied to your comment",
      "body": "I totally agree!",
      "entitytype": "comment",
      "entityid": "01952fd0-...",
      "isread": true,
      "createdat": "2026-02-21T22:00:00Z"
    }
  ],
  "meta": { "total": 47, "page": 1, "limit": 20, "pages": 3 }
}
```

---

### GET /me/notifications/unread-count

Fast badge count for the notification bell icon.

**Auth required:** Yes

**Response `200 OK`:**
```json
{ "data": { "count": 12 } }
```

> Uses `idx_social_notification_unread` partial index — O(1) query.

---

### PATCH /me/notifications/:id/read

Mark a single notification as read.

**Auth required:** Yes  
**Path params:** `id` — notification UUIDv7

**Request body:** None

**Response `204 No Content`**

**Side effects:** `social.notification.isread = TRUE`.

**Errors:**
```json
{ "error": "Notification not found", "code": "NOT_FOUND" }
```

---

### PATCH /me/notifications/read-all

Mark all notifications as read.

**Auth required:** Yes

**Request body:** None

**Response `200 OK`:**
```json
{ "data": { "markedcount": 12 } }
```

**Side effects:** `UPDATE social.notification SET isread = TRUE WHERE userid = $1 AND isread = FALSE`

---

### DELETE /me/notifications/:id

Delete a single notification.

**Auth required:** Yes  
**Path params:** `id` — notification UUIDv7

**Response `204 No Content`**

**Side effects:** `social.notification` row hard-deleted.

---

## 5. Recommendations

Community-submitted "If you liked X, read Y" recommendations, sorted by upvotes.

### GET /comics/:id/recommendations

List recommendations for a comic (i.e. `fromcomicid = :id`), sorted by upvotes descending.

**Auth required:** No  
**Path params:** `id` — comic UUIDv7  
**Query params:** `page`, `limit` (default 10, max 50)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 5000001,
      "fromcomicid": "01952fb0-...",
      "tocomic": {
        "id": "01952fa3-...", "title": "One Piece", "slug": "one-piece",
        "coverurl": "...", "status": "ongoing"
      },
      "submittedby": { "id": "01952fa3-...", "username": "buivan", "avatarurl": null },
      "reason": "Both are long-running action shounen with great world-building.",
      "upvotes": 142,
      "userVote": null,
      "createdat": "2026-02-15T00:00:00Z"
    }
  ],
  "meta": { "total": 8, "page": 1, "limit": 10, "pages": 1 }
}
```

---

### POST /comics/:id/recommendations

Submit a recommendation for a comic.

**Auth required:** Yes  
**Path params:** `id` — source comic UUIDv7

**Request body:**
```json
{
  "tocomicid": "01952fa3-...",
  "reason": "Both are long-running action shounen with great world-building."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `tocomicid` | string | Yes | Must exist. Cannot equal `:id` (no self-recommendation). |
| `reason` | string \| null | No | Max 500 chars |

**Response `201 Created`:** New `ComicRecommendation` object.

**Side effects:** `social.comicrecommendation` row created.

**Errors:**
```json
{ "error": "Cannot recommend a comic to itself", "code": "VALIDATION_ERROR" }
{ "error": "You have already recommended this comic here", "code": "CONFLICT" }
{ "error": "Recommended comic not found", "code": "NOT_FOUND" }
```

---

### POST /recommendations/:id/vote

Upvote or downvote a recommendation.

**Auth required:** Yes  
**Path params:** `id` — recommendation integer ID

**Request body:**
```json
{ "vote": 1 }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `vote` | int | Yes | `1` (upvote) or `-1` (downvote) |

**Response `200 OK`:**
```json
{ "data": { "upvotes": 143, "userVote": 1 } }
```

**Side effects:**
- `social.comicrecommendationvote` upserted
- `social.comicrecommendation.upvotes` updated (delta)
- Cannot vote on own recommendation

**Errors:**
```json
{ "error": "Cannot vote on your own recommendation", "code": "FORBIDDEN" }
```

---

### DELETE /recommendations/:id/vote

Remove vote from a recommendation.

**Auth required:** Yes  
**Path params:** `id` — recommendation integer ID

**Response `200 OK`:**
```json
{ "data": { "upvotes": 142, "userVote": null } }
```

---

### DELETE /recommendations/:id

Delete a recommendation (submitter or admin/mod).

**Auth required:** Yes  
**Path params:** `id` — recommendation integer ID

**Response `204 No Content`**

**Side effects:** `social.comicrecommendation` hard-deleted (cascade deletes votes).

---

## 6. Activity Feed

Pull-based activity feed. Shows events from comics, groups, and users that the caller follows.

### GET /me/feed

Get the current user's personalized activity feed.

**Auth required:** Yes

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `type` | string | — | Filter by event type: `chapter_published` \| `list_updated` \| `comic_updated` \| `user_followed` \| `group_chapter` \| `comic_followed` |
| `before` | string | — | ISO 8601 cursor — events before this time (for pagination) |
| `limit` | int | `20` | Max `50` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952ff0-...",
      "eventtype": "chapter_published",
      "actor": null,
      "entitytype": "chapter",
      "entityid": "01952fa5-...",
      "payload": {
        "comicid": "01952fb0-...",
        "comictitle": "Solo Leveling",
        "coverurl": "https://cdn.yomira.app/covers/solo-leveling.webp",
        "chapternumber": 180.0,
        "chapterlanguage": "en",
        "scanlationgroupname": "MangaPlus (Official)"
      },
      "createdat": "2026-02-22T00:00:00Z"
    },
    {
      "id": "01952ff1-...",
      "eventtype": "list_updated",
      "actor": { "id": "01952fa3-...", "username": "buivan", "avatarurl": null },
      "entitytype": "list",
      "entityid": "01952fc0-...",
      "payload": {
        "listname": "My Top Isekai",
        "addedcomic": { "id": "...", "title": "Mushoku Tensei", "coverurl": "..." }
      },
      "createdat": "2026-02-21T23:30:00Z"
    }
  ],
  "meta": {
    "limit": 20,
    "nextbefore": "2026-02-21T23:30:00Z"
  }
}
```

> Feed uses **cursor-based pagination** (`before` param) instead of offset, to avoid row shifting on new inserts.

**SQL strategy:**
```sql
SELECT * FROM social.feedevent
WHERE entityid IN (
    -- followed comics
    SELECT comicid FROM library.entry WHERE userid = $1
    UNION ALL
    -- followed groups
    SELECT groupid FROM core.scanlationgroupfollow WHERE userid = $1
    UNION ALL
    -- followed users' public list events
    SELECT entityid FROM social.feedevent fe
    WHERE fe.actorid IN (SELECT followingid FROM users.follow WHERE followerid = $1)
)
AND createdat < $2           -- cursor
ORDER BY createdat DESC
LIMIT $3;
```

---

## 7. Forum

### GET /forums

List all forum boards. Site-wide boards have `comicid = null`.

**Auth required:** No

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 100000, "comicid": null,
      "name": "Announcements", "slug": "announcements",
      "description": "Official Yomira announcements.",
      "sortorder": 0, "isarchived": false, "canpost": "admin",
      "threadcount": 12, "postcount": 143
    },
    {
      "id": 100001, "comicid": null,
      "name": "General", "slug": "general",
      "description": "Off-topic chat and general conversation.",
      "sortorder": 1, "isarchived": false, "canpost": "member",
      "threadcount": 2450, "postcount": 31200
    }
  ]
}
```

---

### GET /forums/:slug

Get a single forum board by slug.

**Auth required:** No  
**Path params:** `slug` — e.g. `general`, `one-piece`

**Response `200 OK`:** Single `Forum` object.

**Errors:** `404 NOT_FOUND`

---

### GET /forums/:slug/threads

List threads in a board.

**Auth required:** No  
**Path params:** `slug` — board slug

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `sort` | string | `activity` | `activity` (lastpostedat DESC) \| `new` (createdat DESC) \| `top` (replycount DESC) |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fg0-...", "forumid": 100001,
      "author": { "id": "01952fa3-...", "username": "buivan", "avatarurl": null },
      "title": "What are your top 5 isekai?",
      "ispinned": false, "islocked": false, "isdeleted": false,
      "replycount": 87, "viewcount": 1200,
      "lastpostedat": "2026-02-22T00:00:00Z",
      "lastposter": { "id": "...", "username": "mangafan99" },
      "createdat": "2026-02-20T00:00:00Z"
    }
  ],
  "meta": { "total": 2450, "page": 1, "limit": 20, "pages": 123 }
}
```

> Pinned threads (`ispinned = TRUE`) are always returned first, regardless of sort order.

---

### POST /forums/:slug/threads

Create a new thread in a board.

**Auth required:** Yes (minimum role defined by `forum.canpost`)  
**Path params:** `slug` — board slug

**Request body:**
```json
{
  "title": "What are your top 5 isekai?",
  "body": "I'll start: Mushoku Tensei, Overlord, Re:Zero..."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `title` | string | Yes | 5–500 chars |
| `body` | string | Yes | 1–50 000 chars (first post of the thread) |

**Response `201 Created`:**
```json
{
  "data": {
    "thread": { "id": "01952fg0-...", "title": "...", "replycount": 0 },
    "firstpost": { "id": "01952fh0-...", "body": "I'll start...", "createdat": "..." }
  }
}
```

**Side effects:**
- `social.forumthread` row created
- `social.forumpost` row created (first post, body = request `body`)
- `social.forum.threadcount` incremented
- `social.forum.postcount` incremented

**Errors:**
```json
{ "error": "Insufficient role to post in this board", "code": "FORBIDDEN" }
{ "error": "Forum board not found or archived", "code": "NOT_FOUND" }
```

---

### GET /threads/:id

Get thread detail (metadata only — posts fetched separately).

**Auth required:** No  
**Path params:** `id` — thread UUIDv7

**Response `200 OK`:** Full `ForumThread` object.

**Side effects (async):** `social.forumthread.viewcount` incremented via Redis.

**Errors:** `404 NOT_FOUND`

---

### GET /threads/:id/posts

List posts in a thread, oldest first.

**Auth required:** No  
**Path params:** `id` — thread UUIDv7

**Query params:** `page`, `limit` (default 20, max 100)

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fh0-...", "threadid": "01952fg0-...",
      "author": { "id": "01952fa3-...", "username": "buivan", "avatarurl": null },
      "body": "I'll start: Mushoku Tensei, Overlord...",
      "bodyformat": "markdown",
      "isedited": false, "isdeleted": false, "isapproved": true,
      "upvotes": 12, "downvotes": 0,
      "userVote": null,
      "createdat": "2026-02-20T00:00:00Z", "updatedat": "2026-02-20T00:00:00Z"
    }
  ],
  "meta": { "total": 88, "page": 1, "limit": 20, "pages": 5 }
}
```

---

### POST /threads/:id/posts

Reply to a thread (add a new post).

**Auth required:** Yes (minimum role per board `canpost`)  
**Path params:** `id` — thread UUIDv7

**Request body:**
```json
{
  "body": "My top 5: Solo Leveling, Overlord...",
  "bodyformat": "markdown"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `body` | string | Yes | 1–50 000 chars |
| `bodyformat` | string | No | `markdown` \| `plain`. Default: `markdown` |

**Response `201 Created`:** New `ForumPost` object.

**Side effects:**
- `social.forumpost` row created
- `social.forumthread.replycount` incremented
- `social.forumthread.lastpostedat` + `lastposterid` updated
- `social.forum.postcount` incremented
- `social.notification` created for thread author (type: `comment_reply`)

**Errors:**
```json
{ "error": "Thread is locked", "code": "FORBIDDEN" }
{ "error": "Thread not found", "code": "NOT_FOUND" }
{ "error": "Insufficient role to post in this board", "code": "FORBIDDEN" }
```

---

### PATCH /posts/:id

Edit a forum post.

**Auth required:** Yes (post author only)  
**Path params:** `id` — post UUIDv7

**Request body:**
```json
{ "body": "Edited reply body.", "bodyformat": "markdown" }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `body` | string | Yes | 1–50 000 chars |
| `bodyformat` | string | No | `markdown` \| `plain` |

**Response `200 OK`:** Updated `ForumPost` object (`isedited = true`).

**Errors:**
```json
{ "error": "Post not found or you do not own this post", "code": "FORBIDDEN" }
{ "error": "Cannot edit a deleted post", "code": "VALIDATION_ERROR" }
```

---

### DELETE /posts/:id

Soft-delete a forum post.

**Auth required:** Yes (author or admin/mod)

**Response `204 No Content`**

**Side effects:** `social.forumpost.isdeleted = TRUE`. `social.forumthread.replycount` decremented.

---

### POST /posts/:id/vote

Vote on a forum post.

**Auth required:** Yes  
**Path params:** `id` — post UUIDv7

**Request body:** `{ "vote": 1 }` — `1` or `-1`

**Response `200 OK`:**
```json
{ "data": { "upvotes": 13, "downvotes": 0, "userVote": 1 } }
```

**Side effects:** `social.forumpostvote` upserted. `forumpost.upvotes`/`downvotes` updated.

**Errors:**
```json
{ "error": "Cannot vote on your own post", "code": "FORBIDDEN" }
{ "error": "Vote must be 1 or -1", "code": "VALIDATION_ERROR" }
```

---

### DELETE /posts/:id/vote

Remove vote from a forum post.

**Auth required:** Yes  
**Path params:** `id` — post UUIDv7

**Response `200 OK`:**
```json
{ "data": { "upvotes": 12, "downvotes": 0, "userVote": null } }
```

---

### Admin — Thread & Forum Moderation

#### PATCH /admin/threads/:id/pin

Pin or unpin a thread.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:** `{ "ispinned": true }`

**Response `200 OK`:** `{ "data": { "id": "...", "ispinned": true } }`

---

#### PATCH /admin/threads/:id/lock

Lock or unlock a thread (prevent new replies).

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:** `{ "islocked": true, "reason": "Discussion has derailed" }`

| Field | Type | Required |
|---|---|---|
| `islocked` | bool | Yes |
| `reason` | string | No |

**Response `200 OK`:** `{ "data": { "id": "...", "islocked": true } }`

---

#### DELETE /admin/threads/:id

Soft-delete an entire thread.

**Auth required:** Yes (role: `admin` | `moderator`)

**Request body:** `{ "reason": "Thread contains prohibited content" }`

**Response `204 No Content`**

**Side effects:** `social.forumthread.isdeleted = TRUE`. `social.forum.threadcount` decremented. `system.auditlog` written.

---

#### PATCH /admin/forums/:slug/archive

Archive or restore a forum board.

**Auth required:** Yes (role: `admin`)

**Request body:** `{ "isarchived": true, "reason": "Board consolidated into General" }`

| Field | Type | Required |
|---|---|---|
| `isarchived` | bool | Yes |
| `reason` | string | No |

**Response `200 OK`:** Updated `Forum` object.

**Side effects:** `social.forum.isarchived` updated. Archived boards accept no new threads or posts. `system.auditlog` written.

---

## 8. Reports

### POST /reports

Submit a content moderation report.

**Auth required:** Yes

**Request body:**
```json
{
  "entitytype": "comment",
  "entityid": "01952fd0-...",
  "reason": "spam",
  "details": "This user is posting the same link repeatedly across multiple comics."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `entitytype` | string | Yes | `comic` \| `chapter` \| `comment` \| `forumpost` \| `user` \| `scanlationgroup` |
| `entityid` | string | Yes | Must exist |
| `reason` | string | Yes | `spam` \| `violence` \| `explicit_content` \| `misinformation` \| `copyright` \| `duplicate` \| `low_quality` \| `other` |
| `details` | string \| null | No | Max 2000 chars. Required when `reason = other`. |

**Response `201 Created`:**
```json
{ "data": { "id": "01952fe5-...", "status": "open", "createdat": "2026-02-22T00:12:40Z" } }
```

**Side effects:** `social.report` row created (`status = 'open'`).

**Errors:**
```json
{ "error": "Invalid entity type", "code": "VALIDATION_ERROR" }
{ "error": "Details are required when reason is 'other'", "code": "VALIDATION_ERROR" }
{ "error": "You have already reported this content", "code": "CONFLICT" }
```

---

### GET /admin/reports

List moderation reports (admin/mod queue).

**Auth required:** Yes (role: `admin` | `moderator`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `status` | string | `open` | `open` \| `reviewing` \| `resolved` \| `dismissed` |
| `entitytype` | string | — | Filter by entity type |
| `since` | string | — | ISO 8601 — reports submitted after this time |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fe5-...",
      "reporter": { "id": "01952fa3-...", "username": "buivan" },
      "entitytype": "comment", "entityid": "01952fd0-...",
      "reason": "spam", "details": "Posting the same link repeatedly.",
      "status": "open",
      "resolvedby": null, "resolvedat": null, "resolution": null,
      "createdat": "2026-02-22T00:12:40Z"
    }
  ],
  "meta": { "total": 14, "page": 1, "limit": 50, "pages": 1 }
}
```

---

### PATCH /admin/reports/:id

Update report status and add a resolution note.

**Auth required:** Yes (role: `admin` | `moderator`)  
**Path params:** `id` — report UUIDv7

**Request body:**
```json
{
  "status": "resolved",
  "resolution": "Comment removed. User issued a warning."
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `status` | string | Yes | `reviewing` \| `resolved` \| `dismissed` |
| `resolution` | string \| null | No | Required when `status = resolved` or `dismissed` |

**Response `200 OK`:** Updated `Report` object.

**Side effects:** `social.report.status`, `resolvedby`, `resolvedat`, `resolution` updated. `system.auditlog` written.

**Errors:**
```json
{ "error": "Resolution note is required when resolving or dismissing a report", "code": "VALIDATION_ERROR" }
{ "error": "Report not found", "code": "NOT_FOUND" }
```

---

## 9. Implementation Notes

### Go validation (NOT SQL)

```go
// social.comicrating
score: 1 <= score <= 10

// social.comment
body: 1 <= len(body) <= 10000
comicid XOR chapterid — never both, never neither (Go enforces)
reply depth: replies are always 1 level deep (parentid = root comment ID)
cannot vote on own comment or recommendation

// social.comicrecommendation
fromcomicid != tocomicid (self-recommendation prevention)
reason: len(reason) <= 500 (if not nil)

// social.report
reason: "spam"|"violence"|"explicit_content"|"misinformation"|"copyright"|"duplicate"|"low_quality"|"other"
entitytype: "comic"|"chapter"|"comment"|"forumpost"|"user"|"scanlationgroup"
details required when reason = "other"

// social.forumthread / social.forumpost
title: 5 <= len(title) <= 500
body: 1 <= len(body) <= 50000
bodyformat: "markdown" | "plain"
caller role must meet forum.canpost threshold

// social.forumpostvote / social.commentvote
vote: 1 or -1
cannot vote on own post/comment
```

### Denormalized counter update pattern

```go
// Comment vote — atomic counter update
UPDATE social.comment
SET upvotes   = upvotes   + CASE WHEN $newvote = 1  THEN 1 ELSE 0 END
              - CASE WHEN $oldvote = 1  THEN 1 ELSE 0 END,
    downvotes = downvotes + CASE WHEN $newvote = -1 THEN 1 ELSE 0 END
              - CASE WHEN $oldvote = -1 THEN 1 ELSE 0 END
WHERE id = $commentid;
```

### Caching strategy

| Resource | TTL | Redis key |
|---|---|---|
| `GET /forums` | 10 min | `forums:all` |
| `GET /forums/:slug` | 5 min | `forum:{slug}` |
| `GET /forums/:slug/threads` | 1 min | `forum:{slug}:threads:p{n}:{sort}` |
| `GET /threads/:id` | 1 min | `thread:{id}` |
| `GET /me/notifications/unread-count` | No cache — partial index fast enough | — |
| `GET /me/feed` | No cache (personalized) | — |
| `GET /comics/:id/recommendations` | 2 min | `comic:{id}:recs:p{n}` |

### Auto-generated feed events

| Trigger | `eventtype` | `entitytype` |
|---|---|---|
| Chapter uploaded to a followed comic | `chapter_published` | `chapter` |
| User follows a comic | `comic_followed` | `comic` |
| User follows another user | `user_followed` | `user` |
| Group uploads a chapter | `group_chapter` | `chapter` |
| User updates a public custom list | `list_updated` | `list` |
| Comic metadata updated by admin/mod | `comic_updated` | `comic` |
