# API Reference — Search

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`

> Global conventions — see [API_CONVENTIONS.md](./API_CONVENTIONS.md).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Global search, quick search, per-entity search, autocomplete. |

---

## Table of Contents

1. [Search Architecture](#1-search-architecture)
2. [Global Search](#2-global-search)
3. [Quick Search / Autocomplete](#3-quick-search--autocomplete)
4. [Comic Search](#4-comic-search)
5. [Chapter Search](#5-chapter-search)
6. [Author / Artist Search](#6-author--artist-search)
7. [Tag & Language Search](#7-tag--language-search)
8. [User Search](#8-user-search)
9. [Forum Search](#9-forum-search)
10. [Implementation Notes](#10-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/search` | No | Global search across all entity types |
| `GET` | `/search/quick` | No | Fast autocomplete (< 100ms target) |
| `GET` | `/comics` | No | Advanced comic search with full filter set |
| `GET` | `/chapters` | No | Search chapters by number, language, group |
| `GET` | `/authors` | No | Search authors by name |
| `GET` | `/artists` | No | Search artists by name |
| `GET` | `/tags` | No | List / search tags |
| `GET` | `/languages` | No | List supported languages |
| `GET` | `/users` | No | Search public user profiles |
| `GET` | `/groups` | No | Search scanlation groups |
| `GET` | `/forums/search` | No | Search forum threads and posts |

---

## 1. Search Architecture

Yomira uses **PostgreSQL trigram search** (`pg_trgm`) as the primary search engine for v1.0.0.

```sql
-- Enabled via:
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram indexes on search-heavy columns:
CREATE INDEX idx_core_comic_title_trgm   ON core.comic   USING GIN (title gin_trgm_ops);
CREATE INDEX idx_core_author_name_trgm   ON core.author  USING GIN (name gin_trgm_ops);
CREATE INDEX idx_core_tag_name_trgm      ON core.tag     USING GIN (name gin_trgm_ops);
CREATE INDEX idx_users_account_uname_trgm ON users.account USING GIN (username gin_trgm_ops);
```

**Search strategy:**
| Query type | Method | Notes |
|---|---|---|
| Short string (< 3 chars) | `ILIKE '%q%'` | Trigram needs ≥ 3 chars |
| Normal query (≥ 3 chars) | `similarity(col, q) > 0.3` | Trigram similarity |
| Exact match | `col = q` | Always checked first |
| Alternative names (`namealt`) | `namealt @> ARRAY[$q]` | Array contains check |

**Scale path:** Migrate to Elasticsearch / Meilisearch when comic count exceeds 500K or search latency exceeds 200ms p95.

---

## 2. Global Search

### GET /search

Search across all entity types in a single request.

**Auth required:** No

**Query params:**

| Param | Type | Required | Description |
|---|---|---|---|
| `q` | string | Yes | Search query. Min 1 char, max 200 chars. |
| `type` | string | No | Comma-separated entity types to include: `comic,author,artist,tag,user,group`. Default: all. |
| `limit` | int | No | Max results **per type**. Default `5`, max `20`. |

**Response `200 OK`:**
```json
{
  "data": {
    "query": "solo leveling",
    "results": {
      "comics": {
        "items": [
          {
            "id": "01952fb0-...", "title": "Solo Leveling",
            "slug": "solo-leveling", "type": "comic",
            "coverurl": "https://cdn.yomira.app/covers/01952fb0-.../cover.webp",
            "status": "completed", "contentrating": "safe",
            "score": 0.98
          }
        ],
        "total": 3
      },
      "authors": {
        "items": [
          { "id": 100001, "name": "Chugong", "type": "author", "imageurl": null, "score": 0.72 }
        ],
        "total": 1
      },
      "artists": {
        "items": [
          { "id": 100010, "name": "Kim Jung-hyun", "type": "artist", "imageurl": null, "score": 0.65 }
        ],
        "total": 1
      },
      "tags": { "items": [], "total": 0 },
      "users": { "items": [], "total": 0 },
      "groups": { "items": [], "total": 0 }
    },
    "tookms": 14
  }
}
```

| Response field | Description |
|---|---|
| `score` | Trigram similarity score (0.0–1.0). Results sorted descending. |
| `total` | Total matches found (may exceed `limit`). |
| `tookms` | Server-side query time in milliseconds. |

**Errors:**
```json
{ "error": "Query must be at least 1 character", "code": "VALIDATION_ERROR" }
{ "error": "Query exceeds maximum length of 200 characters", "code": "VALIDATION_ERROR" }
```

---

## 3. Quick Search / Autocomplete

### GET /search/quick

Ultra-fast search for autocomplete dropdowns. Returns only titles/names with no heavy JOINs.

**Auth required:** No  
**Rate limit:** 30 req/min per IP (higher than default — designed for keyup events)  
**Target latency:** < 100ms p95

**Query params:**

| Param | Type | Required | Description |
|---|---|---|---|
| `q` | string | Yes | Min 2 chars, max 100 chars |
| `type` | string | No | `comic` \| `author` \| `artist` \| `tag` \| `group`. Default: `comic`. |
| `limit` | int | No | Default `8`, max `15` |

**Response `200 OK`:**
```json
{
  "data": [
    { "id": "01952fb0-...", "label": "Solo Leveling", "type": "comic", "coverurl": "..." },
    { "id": "01952fb1-...", "label": "Solo Leveling: Ragnarok", "type": "comic", "coverurl": "..." },
    { "id": "01952fb2-...", "label": "Solo Leveling Side Story", "type": "comic", "coverurl": "..." }
  ]
}
```

> No `meta.total` — quick search returns matches only, no count.

**SQL (optimized for speed):**
```sql
SELECT id, title AS label, 'comic' AS type, coverurl
FROM core.comic
WHERE title ILIKE $q || '%'         -- prefix match (fastest, uses B-tree)
   OR title % $q                    -- trigram similarity fallback
   AND deletedat IS NULL
ORDER BY
    CASE WHEN LOWER(title) = LOWER($q) THEN 0    -- exact match first
         WHEN LOWER(title) LIKE LOWER($q) || '%' THEN 1  -- prefix second
         ELSE 2 END,
    followcount DESC,                -- popular comics rank higher
    similarity(title, $q) DESC
LIMIT 8;
```

---

## 4. Comic Search

### GET /comics

Full-featured comic search with all available filters. This is the main discovery endpoint.

**Auth required:** No  
**Rate limit:** 30 req/min per IP

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `title` + `titlealt` |
| `status` | string | — | `ongoing` \| `completed` \| `hiatus` \| `cancelled` \| `unknown`. Multi-value: `status=ongoing,completed` |
| `contentrating` | string | `safe,suggestive` | `safe` \| `suggestive` \| `explicit`. Multi-value. |
| `demographic` | string | — | `shounen` \| `shoujo` \| `seinen` \| `josei`. Multi-value. |
| `tags` | string | — | Comma-separated tag slugs. All tags must match (AND logic). |
| `excludetags` | string | — | Comma-separated tag slugs to exclude. |
| `authors` | string | — | Comma-separated author IDs. |
| `artists` | string | — | Comma-separated artist IDs. |
| `language` | string | — | BCP-47 code — filter by original language. |
| `translatedlang` | string | — | BCP-47 code — has chapters in this language. |
| `year` | int | — | Publication year. |
| `minyear` | int | — | Minimum publication year. |
| `maxyear` | int | — | Maximum publication year. |
| `minrating` | float | — | Minimum Bayesian rating (e.g. `8.0`). |
| `sort` | string | `relevance` | `relevance` \| `latest` \| `oldest` \| `title` \| `rating` \| `follows` \| `views` \| `chapters` |
| `page` | int | `1` | — |
| `limit` | int | `24` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fb0-...", "title": "Solo Leveling", "slug": "solo-leveling",
      "titlealt": ["나 혼자만 레벨업", "Only I Level Up"],
      "coverurl": "https://cdn.yomira.app/covers/01952fb0-.../cover.webp",
      "status": "completed", "contentrating": "safe",
      "demographic": "shounen",
      "year": 2018,
      "ratingbayesian": 9.12, "ratingcount": 58421,
      "followcount": 284021, "viewcount": 10482031, "chaptercount": 179,
      "tags": [
        { "slug": "action", "name": "Action" },
        { "slug": "fantasy", "name": "Fantasy" }
      ],
      "authors": [{ "id": 100001, "name": "Chugong" }],
      "artists": [{ "id": 100010, "name": "Kim Jung-hyun" }],
      "availablelanguages": ["en", "vi", "ja"],
      "userInLibrary": "reading"
    }
  ],
  "meta": { "total": 3, "page": 1, "limit": 24, "pages": 1 }
}
```

> `userInLibrary` — present only for authenticated users; their `library.entry.readingstatus` or `null`.

---

## 5. Chapter Search

### GET /chapters

Search chapters across all comics.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `comicid` | string | — | Filter by comic (commonly used) |
| `language` | string | — | BCP-47 code |
| `groupid` | string | — | Filter by scanlation group |
| `chapternumber` | float | — | Exact chapter number (e.g. `110.5`) |
| `minchapter` | float | — | Min chapter number |
| `maxchapter` | float | — | Max chapter number |
| `isofficial` | bool | — | `true` = official releases only |
| `sort` | string | `chapternumber` | `chapternumber` (asc) \| `createdat` (desc) |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa5-...", "comicid": "01952fb0-...",
      "comic": { "id": "01952fb0-...", "title": "Solo Leveling", "coverurl": "..." },
      "chapternumber": 180.0, "volume": 21,
      "title": null, "language": { "code": "en", "name": "English" },
      "scanlationgroup": { "id": "...", "name": "MangaPlus (Official)", "isofficialpublisher": true },
      "pagecount": 24, "isofficial": true,
      "createdat": "2026-02-22T00:00:00Z",
      "userRead": true
    }
  ],
  "meta": { "total": 179, "page": 1, "limit": 50, "pages": 4 }
}
```

> `userRead` — present only for authenticated users; `true` if this chapter is in `library.chapterread`.

---

## 6. Author / Artist Search

### GET /authors

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `name` + `namealt` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 100001, "name": "Chugong", "namealt": ["추공"], "imageurl": null, "comiccount": 2 }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

---

### GET /artists

Identical to `GET /authors` — replace `author` → `artist` in all field names.

---

## 7. Tag & Language Search

### GET /tags

List or search tags.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Search tag name |
| `category` | string | — | `genre` \| `theme` \| `format` — filter by tag category |
| `sort` | string | `name` | `name` (asc) \| `comiccount` (desc) |
| `limit` | int | `100` | Max `500` |

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 200001, "slug": "action", "name": "Action", "category": "genre", "comiccount": 18420 },
    { "id": 200002, "slug": "fantasy", "name": "Fantasy", "category": "genre", "comiccount": 14200 }
  ],
  "meta": { "total": 84, "page": 1, "limit": 100, "pages": 1 }
}
```

---

### GET /languages

List all supported languages (BCP-47).

**Auth required:** No  
**No query params**

**Response `200 OK`:**
```json
{
  "data": [
    { "id": 1, "code": "en", "name": "English", "nativename": "English" },
    { "id": 2, "code": "ja", "name": "Japanese", "nativename": "日本語" },
    { "id": 3, "code": "vi", "name": "Vietnamese", "nativename": "Tiếng Việt" }
  ]
}
```

> Result is **static / cached** — languages change rarely. Redis TTL: 1 hour.

---

## 8. User Search

### GET /users

Search public user profiles.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `username` and `displayname` |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-...", "username": "buivan", "displayname": "Bui Van Tai",
      "avatarurl": null, "bio": "Manga fan.",
      "followerscount": 42, "followingcount": 18,
      "isFollowing": false
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

> `isFollowing` — authenticated callers only.  
> Banned users (`role = 'banned'`) and deleted users (`deletedat IS NOT NULL`) are excluded.

---

### GET /groups

Search scanlation groups.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | — | Trigram search on `name` |
| `isofficialpublisher` | bool | — | Filter official publisher groups |
| `language` | string | — | BCP-47 — groups that primarily release in this language |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fc5-...", "name": "MangaPlus (Official)", "slug": "mangaplus-official",
      "avatarurl": null, "description": "Official Shueisha scanlation group.",
      "isofficialpublisher": true, "isactive": true,
      "followercount": 84200, "chaptercount": 14200
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20, "pages": 1 }
}
```

---

## 9. Forum Search

### GET /forums/search

Search forum threads and posts.

**Auth required:** No

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | Yes | Full-text search (PostgreSQL `tsvector`). Min 3 chars. |
| `type` | string | `thread` | `thread` \| `post` \| `all` |
| `forumslug` | string | — | Restrict to a specific forum board |
| `authorid` | string | — | Filter by author |
| `since` | string | — | ISO 8601 — results newer than this date |
| `page` | int | `1` | — |
| `limit` | int | `20` | Max `100` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "type": "thread",
      "id": "01952fg0-...",
      "forum": { "slug": "manga", "name": "Manga Discussion" },
      "author": { "id": "01952fa3-...", "username": "buivan" },
      "title": "Solo Leveling ending discussion [SPOILERS]",
      "snippet": "...the final chapter was absolutely...",
      "replycount": 142, "lastpostedat": "2026-02-21T22:00:00Z",
      "createdat": "2026-02-20T00:00:00Z"
    },
    {
      "type": "post",
      "id": "01952fh0-...",
      "thread": { "id": "01952fg0-...", "title": "Solo Leveling ending discussion [SPOILERS]" },
      "forum": { "slug": "manga", "name": "Manga Discussion" },
      "author": { "id": "01952fa3-...", "username": "buivan" },
      "snippet": "...Solo Leveling definitely had the best final arc...",
      "createdat": "2026-02-21T20:00:00Z"
    }
  ],
  "meta": { "total": 28, "page": 1, "limit": 20, "pages": 2 }
}
```

**SQL strategy:**
```sql
-- Full-text search on forum threads
SELECT 'thread' AS type, ft.id, ft.title,
       ts_headline('english', ft.title, websearch_to_tsquery('english', $q)) AS snippet
FROM social.forumthread ft
WHERE ft.searchvector @@ websearch_to_tsquery('english', $q)
  AND ft.isdeleted = FALSE
ORDER BY ts_rank(ft.searchvector, websearch_to_tsquery('english', $q)) DESC;
```

---

## 10. Implementation Notes

### PostgreSQL full-text search for Forum

```sql
-- Add tsvector column to forumthread and forumpost (migration):
ALTER TABLE social.forumthread ADD COLUMN searchvector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', title)) STORED;

ALTER TABLE social.forumpost ADD COLUMN searchvector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', body)) STORED;

CREATE INDEX idx_forumthread_search ON social.forumthread USING GIN (searchvector);
CREATE INDEX idx_forumpost_search   ON social.forumpost   USING GIN (searchvector);
```

### Trigram search tuning

```sql
-- Set similarity threshold (default 0.3 — lower = more results but less precise)
SET pg_trgm.similarity_threshold = 0.3;

-- Query example with score:
SELECT id, title, similarity(title, $q) AS score
FROM core.comic
WHERE title % $q  -- uses GIN trigram index
ORDER BY score DESC
LIMIT 24;
```

### Caching strategy

| Endpoint | TTL | Redis key |
|---|---|---|
| `GET /languages` | 1 hour | `search:languages` |
| `GET /tags` (no query) | 10 min | `search:tags:all` |
| `GET /search/quick?q=...` | 30 sec | `search:quick:{type}:{q}` |
| `GET /comics` (with filters) | 1 min | `search:comics:{hash(params)}` |
| `GET /users?q=...` | No cache | — |
| `GET /forums/search` | No cache | — |

### `GET /comics` filter index usage

| Filter | Index used |
|---|---|
| `q` (title) | `idx_core_comic_title_trgm` (GIN trigram) |
| `status` | `idx_core_comic_status` |
| `tags` | `idx_core_comictag_tagid` (M:N join) |
| `authors` | `idx_core_comicauthor_authorid` (M:N join) |
| `sort=rating` | `idx_core_comic_ratingbayesian` |
| `sort=follows` | `idx_core_comic_followcount` |
| Multi-filter combo | Planner uses most selective index first |
