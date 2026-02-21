# API Reference — File Upload & Media

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Source:** Go service + Object Storage (Cloudflare R2 / AWS S3 / MinIO)

> Global conventions — see [API_CONVENTIONS.md](./API_CONVENTIONS.md).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Presigned upload, avatar, comic covers, chapter pages, art gallery. |

---

## Table of Contents

1. [Upload Architecture](#1-upload-architecture)
2. [Common Types](#2-common-types)
3. [Presigned Upload](#3-presigned-upload)
4. [Avatar Upload](#4-avatar-upload)
5. [Comic Cover Upload](#5-comic-cover-upload)
6. [Comic Art Gallery Upload](#6-comic-art-gallery-upload)
7. [Chapter Pages Upload](#7-chapter-pages-upload)
8. [Scanlation Group Avatar Upload](#8-scanlation-group-avatar-upload)
9. [Implementation Notes](#9-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/upload/presign` | Yes | Request a presigned upload URL |
| `POST` | `/upload/confirm` | Yes | Confirm upload complete and register file |
| `PUT` | `/me/avatar` | Yes | Upload / replace user avatar (multipart) |
| `DELETE` | `/me/avatar` | Yes | Remove user avatar |
| `PUT` | `/admin/comics/:id/cover` | admin/mod | Upload / replace comic cover |
| `DELETE` | `/admin/comics/:id/cover` | admin/mod | Remove comic cover |
| `POST` | `/admin/comics/:id/art` | admin/mod | Upload an art gallery image |
| `DELETE` | `/admin/comics/:id/art/:artId` | admin/mod | Remove an art gallery image |
| `POST` | `/admin/chapters/:id/pages/bulk` | admin/mod/group | Bulk upload pages for a chapter |
| `POST` | `/admin/chapters/:id/pages` | admin/mod/group | Upload a single page |
| `DELETE` | `/admin/chapters/:id/pages/:pageNumber` | admin | Delete a specific page |
| `POST` | `/groups/:id/avatar` | Yes (leader) | Upload / replace group avatar |
| `DELETE` | `/groups/:id/avatar` | Yes (leader) | Remove group avatar |

---

## 1. Upload Architecture

Yomira uses a **presigned URL** (direct-to-storage) upload pattern to avoid routing large files through Go:

```
Client               Go API                 Object Storage (R2/S3)
  │                       │                              │
  │  POST /upload/presign │                              │
  │ ──────────────────▶   │                              │
  │                       │  GeneratePresignedPutURL()   │
  │                       │ ────────────────────────────▶│
  │                       │       presigned PUT URL      │
  │                       │ ◀─────────────────────────── │
  │  { uploadUrl, key }   │                              │
  │ ◀──────────────────   │                              │
  │                       │                              │
  │  PUT {uploadUrl}      │                              │
  │ ────────────────────────────────────────────────────▶│
  │  200 OK               │                              │
  │ ◀─────────────────────────────────────────────────── │
  │                       │                              │
  │ POST /upload/confirm  │                              │
  │ ──────────────────▶   │  Verify + register file      │
  │  { MediaFile }        │ ────────────────────────────▶│
  │ ◀──────────────────   │                              │
```

**Simpler endpoints** (`PUT /me/avatar`) use **multipart form** directly to Go — Go proxies to storage. Used for small, user-facing uploads.

---

## 2. Common Types

### `PresignedUploadResponse`
```typescript
{
  uploadUrl: string       // PUT to this URL with raw file bytes
  publicUrl: string       // Final CDN URL after upload completes
  key: string             // Object storage key (use in POST /upload/confirm)
  expiresAt: string       // Presigned URL expiry (15 min from issuance)
}
```

### `MediaFile`
```typescript
{
  id: string
  uploaderid: string
  entitytype: string | null   // "comic_cover" | "comic_art" | "chapter_page" | "avatar" | "group_avatar"
  entityid: string | null
  storagekey: string          // Internal object storage key
  publicurl: string           // Full CDN URL
  format: "webp" | "jpeg" | "png"
  width: number; height: number; filesizebytes: number
  createdat: string
}
```

### CDN URL pattern

```
https://cdn.yomira.app/covers/{comicid}/cover.webp
https://cdn.yomira.app/covers/{comicid}/cover-hd.webp
https://cdn.yomira.app/pages/{chapterid}/p001.webp
https://cdn.yomira.app/pages/{chapterid}/hd/p001.webp
https://cdn.yomira.app/avatars/{userid}/avatar.webp
https://cdn.yomira.app/art/{comicid}/art-{artid}.webp
```

---

## 3. Presigned Upload

### POST /upload/presign

Request a presigned PUT URL for direct-to-storage upload.

**Auth required:** Yes

**Request body:**
```json
{
  "entitytype": "comic_cover",
  "entityid": "01952fb0-...",
  "filename": "cover.jpg",
  "contenttype": "image/jpeg",
  "filesizebytes": 204800
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `entitytype` | string | Yes | `comic_cover` \| `comic_art` \| `chapter_page` \| `avatar` \| `group_avatar` |
| `entityid` | string | Yes | Must exist and caller must have write access |
| `filename` | string | Yes | Used for extension detection |
| `contenttype` | string | Yes | `image/jpeg` \| `image/png` \| `image/webp` |
| `filesizebytes` | int | Yes | Must not exceed entity-type limit (see §9) |

**Response `200 OK`:**
```json
{
  "data": {
    "uploadUrl": "https://r2.internal/bucket/covers/01952fb0-cover-abc123.jpg?X-Amz-Signature=...",
    "publicUrl": "https://cdn.yomira.app/covers/01952fb0-.../cover.webp",
    "key": "covers/01952fb0-cover-abc123.jpg",
    "expiresAt": "2026-02-22T00:50:28Z"
  }
}
```

**Errors:**
```json
{ "error": "File size exceeds maximum allowed for comic_cover (5MB)", "code": "VALIDATION_ERROR" }
{ "error": "Content type not allowed", "code": "VALIDATION_ERROR" }
{ "error": "Insufficient permission to upload to this entity", "code": "FORBIDDEN" }
```

---

### POST /upload/confirm

Confirm upload completed. Registers file in DB and optionally attaches to entity.

**Auth required:** Yes

**Request body:**
```json
{
  "key": "covers/01952fb0-cover-abc123.jpg",
  "entitytype": "comic_cover",
  "entityid": "01952fb0-...",
  "attach": true
}
```

**Response `200 OK`:** `MediaFile` object.

**Side effects:**
- File existence verified (`HeadObject`)
- Image dimensions read; WebP conversion applied if needed
- `core.mediafile` row created
- If `attach = true`: entity updated (e.g. `core.comic.coverurl`)
- Previous orphaned files queued for deletion

**Errors:**
```json
{ "error": "File not found in object storage. Did the upload complete?", "code": "NOT_FOUND" }
```

---

## 4. Avatar Upload

### PUT /me/avatar

Upload or replace user avatar (multipart, proxied via Go).

**Auth required:** Yes  
**Content-Type:** `multipart/form-data`  
**Form field:** `file`

| Constraint | Value |
|---|---|
| Max size | 2 MB |
| Accepted formats | JPEG, PNG, WebP, GIF |
| Output | WebP 200×200 (square crop, center) |

**Response `200 OK`:**
```json
{ "data": { "avatarurl": "https://cdn.yomira.app/avatars/01952fa3-.../avatar.webp" } }
```

**Side effects:** Resized + converted to WebP 200×200. `users.account.avatarurl` updated.

---

### DELETE /me/avatar

**Auth required:** Yes  
**Response `204 No Content`**  
**Side effects:** `users.account.avatarurl = NULL`. File queued for deletion.

---

## 5. Comic Cover Upload

### PUT /admin/comics/:id/cover

**Auth required:** Yes (role: `admin` | `moderator`)  
**Content-Type:** `multipart/form-data`  
**Form field:** `file`

| Constraint | Value |
|---|---|
| Max size | 5 MB |
| Accepted formats | JPEG, PNG, WebP |
| Output | WebP 420×600 standard + 840×1200 HD variant |

**Response `200 OK`:**
```json
{
  "data": {
    "coverurl": "https://cdn.yomira.app/covers/01952fb0-.../cover.webp",
    "coverurlhd": "https://cdn.yomira.app/covers/01952fb0-.../cover-hd.webp"
  }
}
```

**Side effects:** `core.comic.coverurl` + `coverurlhd` updated. `system.auditlog` written.

---

### DELETE /admin/comics/:id/cover

**Auth required:** Yes (role: `admin` | `moderator`)  
**Response `204 No Content`**  
**Side effects:** `core.comic.coverurl = NULL`. File queued for deletion.

---

## 6. Comic Art Gallery Upload

### POST /admin/comics/:id/art

**Auth required:** Yes (role: `admin` | `moderator`)  
**Content-Type:** `multipart/form-data`

**Form fields:**

| Field | Required | Notes |
|---|---|---|
| `file` | Yes | Max 10 MB, JPEG/PNG/WebP |
| `caption` | No | Max 500 chars |
| `sortorder` | No | Integer ≥ 0, default 0 |

**Response `201 Created`:**
```json
{
  "data": {
    "id": "01953210-...", "comicid": "01952fb0-...",
    "imageurl": "https://cdn.yomira.app/art/01952fb0-.../art-001.webp",
    "caption": "Volume 1 Special Colour Spread",
    "sortorder": 0, "width": 1920, "height": 1080,
    "filesizebytes": 412800, "createdat": "2026-02-22T00:35:28Z"
  }
}
```

**Side effects:** `core.comicart` row created. `system.auditlog` written.

**Errors:**
```json
{ "error": "Comic art gallery is full (100 images maximum)", "code": "LIMIT_EXCEEDED" }
```

---

### DELETE /admin/comics/:id/art/:artId

**Auth required:** Yes (role: `admin` | `moderator`)  
**Response `204 No Content`**  
**Side effects:** `core.comicart` hard-deleted. File queued for deletion.

---

## 7. Chapter Pages Upload

### POST /admin/chapters/:id/pages/bulk

Bulk upload multiple pages (zip archive or multiple form files).

**Auth required:** Yes (role: `admin` | `moderator` | group `leader/moderator`)  
**Content-Type:** `multipart/form-data`

**Form fields:**

| Field | Required | Notes |
|---|---|---|
| `files` | Yes | Multiple images OR single `.zip`. Named `p001.jpg`, `p002.jpg`... for auto-ordering. |
| `hdquality` | No | `true` (default) = generate HD variant |

| Constraint | Value |
|---|---|
| Max per page | 10 MB |
| Max pages | 200 per request |
| Max total size | 500 MB |
| Accepted formats | JPEG, PNG, WebP |
| Output | WebP (original dimensions, max 2000px longest side) |

**Response `202 Accepted`** (async processing):
```json
{
  "data": {
    "jobid": "01953220-...", "chapterid": "01952fa5-...",
    "status": "queued", "pagecount": 24,
    "message": "Pages are being processed. Use GET /admin/chapters/:id to check syncstate."
  }
}
```

**Processing pipeline (async):**
1. Validate each image
2. Convert to WebP + generate HD variant
3. Upload to storage: `pages/{chapterid}/p{NNN}.webp` + `hd/p{NNN}.webp`
4. Insert `core.page` rows
5. `UPDATE core.chapter SET pagecount = $n, syncstate = 'synced'`
6. `UPDATE library.entry SET hasnew = TRUE` for all followers

---

### POST /admin/chapters/:id/pages

Upload a single page.

**Auth required:** Yes (role: `admin` | `moderator` | group `leader/moderator`)

**Form fields:** `file`, `pagenumber` (required int), `replace` (bool, default `false`)

**Response `201 Created`:**
```json
{
  "data": {
    "id": "01953230-...", "pagenumber": 5,
    "imageurl": "https://cdn.yomira.app/pages/01952fa5-.../p005.webp",
    "imageurlhd": "https://cdn.yomira.app/pages/01952fa5-.../hd/p005.webp",
    "format": "webp", "width": 784, "height": 1200, "filesizebytes": 89240
  }
}
```

**Errors:**
```json
{ "error": "Page 5 already exists. Set replace=true to overwrite.", "code": "CONFLICT" }
{ "error": "Chapter is locked", "code": "FORBIDDEN" }
```

---

### DELETE /admin/chapters/:id/pages/:pageNumber

**Auth required:** Yes (role: `admin`)  
**Response `204 No Content`**  
**Side effects:** `core.page` hard-deleted. `core.chapter.pagecount` decremented. File queued for deletion.

---

## 8. Scanlation Group Avatar Upload

### POST /groups/:id/avatar

**Auth required:** Yes (group `leader` or `admin`)  
**Content-Type:** `multipart/form-data`  
**Form field:** `file`

| Constraint | Value |
|---|---|
| Max size | 2 MB |
| Accepted formats | JPEG, PNG, WebP |
| Output | WebP 200×200 (square crop) |

**Response `200 OK`:**
```json
{ "data": { "avatarurl": "https://cdn.yomira.app/groups/01952fc5-.../avatar.webp" } }
```

---

### DELETE /groups/:id/avatar

**Auth required:** Yes (group `leader` or `admin`)  
**Response `204 No Content`**

---

## 9. Implementation Notes

### File size limits by entity type

| Entity type | Max size | Output format | Dimensions |
|---|---|---|---|
| `avatar` (user/group) | 2 MB | WebP | 200×200 square crop |
| `comic_cover` | 5 MB | WebP | 420×600 + HD 840×1200 |
| `comic_art` | 10 MB | WebP | Original, max 2000px |
| `chapter_page` | 10 MB | WebP | Original, max 2000px |
| Bulk upload total | 500 MB | WebP | — |

### Image processing (libvips via govips)

```go
img, _ := vips.NewImageFromBuffer(input)
img.Thumbnail(maxSide, maxSide, vips.InterestingAttention)  // resize
img.SmartCrop(w, h, vips.InterestingAttention)              // square crop
ep := vips.NewWebpExportParams()
ep.Quality = 85
buf, _, _ := img.ExportWebp(ep)
```

### Storage key convention

```
avatars/{userid}/avatar.webp
covers/{comicid}/cover.webp  |  covers/{comicid}/cover-hd.webp
art/{comicid}/art-{artid}.webp
pages/{chapterid}/p{NNN}.webp  |  pages/{chapterid}/hd/p{NNN}.webp
```

### Environment variables

| Variable | Example | Description |
|---|---|---|
| `S3_BUCKET` | `yomira-media` | Object storage bucket |
| `S3_ENDPOINT` | `https://xyz.r2.cloudflarestorage.com` | R2/S3 endpoint |
| `S3_REGION` | `auto` | R2: `auto`; AWS: `ap-southeast-1` |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | `...` | Credentials |
| `CDN_BASE_URL` | `https://cdn.yomira.app` | Public CDN prefix |
| `PRESIGN_TTL` | `900` | Presigned URL TTL seconds (15 min default) |
