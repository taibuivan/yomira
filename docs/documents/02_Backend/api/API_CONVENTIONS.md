# API Conventions — Global Reference

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Applies to:** All `/api/v1/*` endpoints

---

## Table of Contents

1. [Base URL & Versioning](#1-base-url--versioning)
2. [Authentication](#2-authentication)
3. [Request Format](#3-request-format)
4. [Response Envelope](#4-response-envelope)
5. [Error Format & Error Codes](#5-error-format--error-codes)
6. [Pagination](#6-pagination)
7. [Date & ID Formats](#7-date--id-formats)
8. [Rate Limiting](#8-rate-limiting)
9. [CORS](#9-cors)
10. [HTTP Status Code Reference](#10-http-status-code-reference)

---

## 1. Base URL & Versioning

```
Production:  https://api.yomira.app/api/v1
Staging:     https://api.staging.yomira.app/api/v1
Development: http://localhost:8080/api/v1
```

### Versioning policy

- Current version: **v1**
- Version is embedded in the URL path: `/api/v1/`
- Breaking changes → new version (`/api/v2/`). Both versions supported for **6 months** after a new version ships.
- Non-breaking additions (new optional fields, new endpoints) — added silently to existing version.
- **Deprecation notice:** Sent via `Deprecation` and `Sunset` response headers at least 3 months before removal.

```
Deprecation: true
Sunset: Sat, 01 Jan 2027 00:00:00 GMT
Link: </api/v2/auth/login>; rel="successor-version"
```

---

## 2. Authentication

### Access Token (JWT RS256)

Most endpoints require a Bearer token in the `Authorization` header:

```http
Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Token properties:**
| Property | Value |
|---|---|
| Algorithm | `RS256` |
| Lifetime | 15 minutes (`exp` claim) |
| Payload claims | `sub` (userID), `role`, `iat`, `exp`, `jti` |
| Key rotation | Public key served at `GET /api/v1/.well-known/jwks.json` |

### Refresh Token

Long-lived token (default 30 days, configurable via `auth.session_ttl_days` setting).  
Stored as `tokenhash` (SHA-256 of raw token) in `users.session`.  
Exchange for a new access token via `POST /auth/token/refresh`.

### Role Hierarchy

```
admin  >  moderator  >  member  >  banned
```

Endpoints marked `admin/mod` accept both `admin` and `moderator` roles.

### Unauthenticated Requests

Endpoints marked **"No"** for auth are publicly accessible.  
For those endpoints, if a valid `Authorization` header is included anyway, the caller will receive **personalized responses** (e.g. `userVote`, `userRating`, library status).

---

## 3. Request Format

### Content-Type

```http
Content-Type: application/json
```

All request bodies must be valid JSON. `multipart/form-data` is only used for file uploads (see UPLOAD_API.md).

### Encoding

UTF-8 required for all text fields. Emoji supported.

### Empty body

For endpoints with no required body (e.g. `DELETE`, `POST` with no params), send an empty body or omit entirely — both are accepted.

### Field names

All field names use **camelCase** (JSON) mapping to **lowercase** (Postgres columns).

```json
{ "readingstatus": "reading" }    ✅ matches Postgres column
{ "readingStatus": "reading" }    ❌ not accepted
```

---

## 4. Response Envelope

All responses share a consistent envelope:

### Success response

```json
{
  "data": { ... } | [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 20,
    "pages": 5
  }
}
```

| Field | When present |
|---|---|
| `data` | Always — single object or array |
| `meta` | Only on paginated list responses (contains `total`, `page`, `limit`, `pages`) |

### No-content response

`204 No Content` — empty body, no envelope.

### Created response

`201 Created` — envelope with `data` = newly created resource.

---

## 5. Error Format & Error Codes

### Error envelope

```json
{
  "error": "Human-readable message suitable for logging",
  "code": "MACHINE_READABLE_CODE",
  "details": [ ... ]
}
```

| Field | Description |
|---|---|
| `error` | Short English description. Not for display to end users. |
| `code` | Stable machine-readable constant. Frontend switches on this. |
| `details` | Optional array — present for validation errors with per-field context. |

### Validation error (multi-field)

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": [
    { "field": "email", "message": "Must be a valid email address" },
    { "field": "password", "message": "Must be at least 8 characters" }
  ]
}
```

### Complete error code table

| Code | HTTP Status | Description | Common Triggers |
|---|---|---|---|
| `VALIDATION_ERROR` | `400` | Request body or params failed validation | Missing required field, out-of-range value, invalid format |
| `UNAUTHORIZED` | `401` | Missing or invalid auth token | No `Authorization` header, expired JWT, revoked session |
| `TOKEN_EXPIRED` | `401` | Auth token has expired | JWT `exp` in the past |
| `TOKEN_INVALID` | `401` | Token signature invalid or tampered | JWT RS256 signature mismatch |
| `FORBIDDEN` | `403` | Authenticated but insufficient role | Member accessing admin endpoint, editing another user's resource |
| `NOT_FOUND` | `404` | Resource does not exist | Wrong ID, soft-deleted resource |
| `CONFLICT` | `409` | Unique constraint violation | Duplicate email, comic already in library |
| `LIMIT_EXCEEDED` | `422` | Business logic limit hit | Over 500 items in list, over 100 custom lists |
| `RATE_LIMITED` | `429` | Too many requests | Exceeds per-IP or per-user rate limit |
| `INTERNAL_ERROR` | `500` | Unexpected server error | Database error, unhandled exception |
| `SERVICE_UNAVAILABLE` | `503` | Service temporarily down | Maintenance mode, DB connection pool exhausted |

---

## 6. Pagination

### Offset pagination (most endpoints)

**Request:**
```
GET /api/v1/comics?page=2&limit=24
```

**Response `meta`:**
```json
{
  "meta": {
    "total": 52000,
    "page": 2,
    "limit": 24,
    "pages": 2167
  }
}
```

| Param | Type | Default | Max | Notes |
|---|---|---|---|---|
| `page` | int | `1` | — | 1-indexed |
| `limit` | int | varies | varies | Max limit varies by endpoint — see each endpoint |

> **When to use offset:** Navigable pages (comics list, search results, admin tables).

### Cursor pagination (feed, logs)

Used for **time-series data** where rows can be inserted mid-navigation.

**Request:**
```
GET /api/v1/me/feed?before=2026-02-22T00:00:00Z&limit=20
```

**Response `meta`:**
```json
{
  "meta": {
    "limit": 20,
    "nextbefore": "2026-02-21T22:00:00Z"
  }
}
```

Pass `nextbefore` as the `before` param in the next request. `nextbefore = null` means you've reached the end.

> **When to use cursor:** `GET /me/feed`, `GET /admin/crawler/jobs/:id/logs`

---

## 7. Date & ID Formats

### Timestamps

All timestamps are **ISO 8601 with UTC timezone**:

```
2026-02-22T00:35:28Z        ✅ correct
2026-02-22T07:35:28+07:00   ✅ also accepted (converted to UTC server-side)
2026-02-22 00:35:28         ❌ not accepted (missing T and timezone)
```

### UUIDs (UUIDv7)

Primary keys for user-facing resources (comics, chapters, users, etc.) are **UUIDv7**:

```
01952fa3-a1b2-7000-8000-abcdef123456
```

Properties:
- **Time-ordered** — newer rows sort to end of B-tree naturally
- **Globally unique** — safe to generate at application layer
- **Non-guessable** — random bits in lower 62 bits
- **URL-safe** — lowercase hex, hyphens

### Integer IDs

Some internal/reference tables use integer sequences (e.g. `core.language.id`, `social.forum.id`, `crawler.source.id`). These are noted per-endpoint.

---

## 8. Rate Limiting

Rate limits are enforced at the **API gateway / Go middleware layer** using Redis sliding windows.

### Default limits

| Scope | Limit | Window | Applied to |
|---|---|---|---|
| Global (unauthenticated) | 60 req | 1 minute | Per IP |
| Global (authenticated) | 300 req | 1 minute | Per user |
| Auth endpoints (`/auth/*`) | 10 req | 15 minutes | Per IP |
| Write endpoints (`POST/PATCH/DELETE`) | 60 req | 1 minute | Per user |
| Email endpoints | 3 req | 15 minutes | Per email address |
| Upload endpoints | 10 req | 1 minute | Per user |
| Search endpoints | 30 req | 1 minute | Per IP / user |

### Rate limit response headers

Every response includes:

```http
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 247
X-RateLimit-Reset: 1740183328
```

When rate limit is exceeded (`429 Too Many Requests`):

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 42
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1740183370
Content-Type: application/json

{
  "error": "Too many requests. Please try again in 42 seconds.",
  "code": "RATE_LIMITED"
}
```

| Header | Type | Description |
|---|---|---|
| `X-RateLimit-Limit` | int | Max requests allowed in the window |
| `X-RateLimit-Remaining` | int | Requests remaining in current window |
| `X-RateLimit-Reset` | int | Unix timestamp when the window resets |
| `Retry-After` | int | Seconds to wait before retrying (only on 429) |

---

## 9. CORS

Cross-Origin Resource Sharing is configured at the Go middleware layer.

### Allowed origins (production)

```
https://yomira.app
https://www.yomira.app
https://admin.yomira.app
```

### Allowed origins (development)

```
http://localhost:3000
http://localhost:5173
http://127.0.0.1:3000
```

### CORS headers (all responses)

```http
Access-Control-Allow-Origin: https://yomira.app
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
Access-Control-Expose-Headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
Access-Control-Max-Age: 86400
```

### Preflight request

```http
OPTIONS /api/v1/comics
Host: api.yomira.app
Origin: https://yomira.app
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Authorization, Content-Type

→ 204 No Content
```

---

## 10. HTTP Status Code Reference

| Status | Meaning | When returned |
|---|---|---|
| `200 OK` | Success with body | `GET`, `PATCH`, `PUT` responses |
| `201 Created` | Resource created | `POST` that creates a new resource |
| `202 Accepted` | Async operation queued | Batch job triggers |
| `204 No Content` | Success, no body | `DELETE`, some `PATCH` (e.g. mark read) |
| `302 Found` | Redirect | Email token confirmation |
| `400 Bad Request` | Malformed request / validation failed | Invalid JSON, missing required field |
| `401 Unauthorized` | Not authenticated | Missing/expired/invalid token |
| `403 Forbidden` | Authenticated but not authorized | Insufficient role, not resource owner |
| `404 Not Found` | Resource not found | Wrong ID, soft-deleted |
| `409 Conflict` | State conflict | Unique constraint violation, duplicate entry |
| `422 Unprocessable Entity` | Business logic limit | Over feature limit (500 list items, etc.) |
| `429 Too Many Requests` | Rate limited | See §8 |
| `500 Internal Server Error` | Unexpected error | Bug, DB error — always logged |
| `503 Service Unavailable` | Maintenance / overload | `site.maintenance_mode = true` |

---

## Additional Headers

### Request ID

Every request is assigned a unique **correlation ID** for tracing:

```http
X-Request-ID: 01953abc-...   ← set by client (optional)
```

If not provided, Go generates one. The ID is echoed in the response:

```http
X-Request-ID: 01953abc-...
```

Use this ID when reporting issues — it links the request to server logs.

### Content negotiation

Currently only `application/json` is supported. Do not send `Accept: text/html` — always send:

```http
Accept: application/json
```
