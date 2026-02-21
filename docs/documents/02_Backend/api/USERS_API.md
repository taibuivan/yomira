 # API Reference — Users Domain

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-21  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`  
> **Source schema:** `10_USERS/USERS.sql`

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-21 | Initial release. All authentication, profile, session, follow, and preferences endpoints. |

---

## Table of Contents

1. [Global Conventions](#global-conventions)
2. [Common Types](#common-types)
3. [Auth — Registration & Login](#1-auth--registration--login)
4. [Auth — OAuth](#2-auth--oauth)
5. [Sessions](#3-sessions)
6. [User Profile](#4-user-profile)
7. [Follow Graph](#5-follow-graph)
8. [Reading Preferences](#6-reading-preferences)
9. [Admin — User Management](#7-admin--user-management)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/register` | No | Create a new account |
| `POST` | `/auth/login` | No | Email + password login |
| `POST` | `/auth/logout` | Yes | Revoke current session |
| `POST` | `/auth/refresh` | No (cookie) | Rotate refresh token, issue new access token |
| `POST` | `/auth/verify-email` | No | Confirm email with verification token |
| `POST` | `/auth/forgot-password` | No | Request password reset email |
| `POST` | `/auth/reset-password` | No | Set new password with reset token |
| `POST` | `/auth/change-password` | Yes | Change password when logged in |
| `GET` | `/auth/oauth/:provider` | No | Initiate OAuth2 flow (redirect to provider) |
| `GET` | `/auth/oauth/:provider/callback` | No | OAuth2 callback — create/link account |
| `DELETE` | `/auth/oauth/:provider` | Yes | Unlink an OAuth provider |
| `GET` | `/auth/oauth/linked` | Yes | List linked OAuth providers |
| `GET` | `/me/sessions` | Yes | List active sessions |
| `DELETE` | `/me/sessions/:id` | Yes | Revoke a specific session |
| `DELETE` | `/me/sessions` | Yes | Revoke all other sessions |
| `GET` | `/users/:id` | No | Public profile of any user |
| `GET` | `/me` | Yes | Current user's full private profile |
| `PATCH` | `/me` | Yes | Update profile (username, bio, website…) |
| `POST` | `/me/avatar` | Yes | Upload avatar image |
| `DELETE` | `/me` | Yes | Soft-delete own account |
| `POST` | `/users/:id/follow` | Yes | Follow a user |
| `DELETE` | `/users/:id/follow` | Yes | Unfollow a user |
| `GET` | `/users/:id/followers` | No | List followers of a user |
| `GET` | `/users/:id/following` | No | List users that this user follows |
| `GET` | `/me/followers` | Yes | List my followers |
| `GET` | `/me/following` | Yes | List who I follow |
| `GET` | `/me/preferences` | Yes | Get reader UI preferences |
| `PUT` | `/me/preferences` | Yes | Save reader UI preferences (full upsert) |
| `GET` | `/admin/users` | admin | List all users with filters |
| `GET` | `/admin/users/:id` | admin | Full account detail (admin view) |
| `PATCH` | `/admin/users/:id/role` | admin | Change user role |
| `PATCH` | `/admin/users/:id/suspend` | admin | Suspend / restore account |
| `DELETE` | `/admin/users/:id` | admin | Soft-delete user account |

---

## Global Conventions

### Authentication

All protected endpoints require a JWT access token in the `Authorization` header:
```
Authorization: Bearer <access_token>
```

Access tokens expire after **15 minutes**. Use [POST /auth/refresh](#post-authrefresh) with the `refresh_token` HttpOnly cookie to obtain a new one.

### Response Envelope

**Success:**
```json
{
  "data": { ... }
}
```

**Paginated success:**
```json
{
  "data": [ ... ],
  "meta": {
    "total": 120,
    "page": 1,
    "limit": 20,
    "pages": 6
  }
}
```

**Error:**
```json
{
  "error": "Human-readable message",
  "code": "MACHINE_READABLE_CODE"
}
```

### Error Codes

| HTTP | Code | Meaning |
|---|---|---|
| `400` | `VALIDATION_ERROR` | Request failed field validation |
| `400` | `INVALID_CREDENTIALS` | Wrong email or password |
| `400` | `WEAK_PASSWORD` | Password does not meet policy |
| `401` | `UNAUTHORIZED` | Missing or expired token |
| `401` | `TOKEN_EXPIRED` | Access token expired — refresh it |
| `403` | `FORBIDDEN` | Authenticated but insufficient role |
| `403` | `ACCOUNT_SUSPENDED` | `isactive = FALSE` |
| `403` | `ACCOUNT_DELETED` | `deletedat IS NOT NULL` |
| `403` | `EMAIL_NOT_VERIFIED` | `isverified = FALSE` (write actions blocked) |
| `404` | `NOT_FOUND` | Resource does not exist |
| `409` | `DUPLICATE_EMAIL` | Email already registered |
| `409` | `DUPLICATE_USERNAME` | Username already taken |
| `409` | `ALREADY_FOLLOWING` | Follow relation already exists |
| `429` | `RATE_LIMITED` | Too many requests |
| `500` | `INTERNAL_ERROR` | Unexpected server error |

### Pagination Query Parameters

| Param | Type | Default | Max | Description |
|---|---|---|---|---|
| `page` | int | `1` | — | Page number (1-based) |
| `limit` | int | `20` | `100` | Items per page |

### Role Values (`users.account.role`)

| Value | Description |
|---|---|
| `admin` | Full system access |
| `moderator` | Content moderation tools |
| `member` | Standard user (default) |
| `banned` | Cannot perform any write action |

---

## Common Types

### `UserPublic` — Safe public representation

```typescript
{
  id: string            // UUIDv7
  username: string
  displayname: string | null
  avatarurl: string | null
  bio: string | null
  website: string | null
  isverified: boolean
  followercount: number  // computed: COUNT(*) FROM users.follow WHERE followingid = id
  followingcount: number // computed: COUNT(*) FROM users.follow WHERE followerid = id
  createdat: string      // ISO 8601
}
```

### `UserPrivate` — Full self profile (only returned for /me)

```typescript
{
  id: string
  username: string
  email: string
  displayname: string | null
  avatarurl: string | null
  bio: string | null
  website: string | null
  role: "admin" | "moderator" | "member" | "banned"
  isverified: boolean
  isactive: boolean
  lastloginat: string | null  // ISO 8601
  followercount: number
  followingcount: number
  oauthproviders: string[]    // e.g. ["google", "discord"]
  createdat: string
  updatedat: string
}
```

### `SessionInfo`

```typescript
{
  id: number
  devicename: string | null
  ipaddress: string | null
  createdat: string     // ISO 8601
  expiresat: string     // ISO 8601
  iscurrent: boolean    // true if this is the calling session
}
```

### `ReadingPreference`

```typescript
{
  readingmode: "ltr" | "rtl" | "vertical" | "webtoon"
  pagefit: "width" | "height" | "original" | "stretch"
  doublepageon: boolean
  showpagebar: boolean
  preloadpages: number   // 1–10
  datasaver: boolean
  hidensfw: boolean
  hidelanguages: string[]  // BCP-47 codes, e.g. ["en", "ja"]
  updatedat: string
}
```

---

## 1. Auth — Registration & Login

### POST /auth/register

Create a new account.

**Auth required:** No  
**Rate limit:** 5 requests/min per IP

**Request body:**
```json
{
  "username": "buivan",
  "email": "tai.buivan.jp@gmail.com",
  "password": "SuperSecret123!"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `username` | string | Yes | 3–64 chars. `[a-zA-Z0-9_-]` only. Case-insensitive unique. |
| `email` | string | Yes | Valid email format. Max 254 chars. |
| `password` | string | Yes | Min 8 chars. At least 1 upper, 1 lower, 1 digit. Max 72 chars (bcrypt limit). |

**Response `201 Created`:**
```json
{
  "data": {
    "id": "01952fa3-3f1e-7abc-b12e-1234567890ab",
    "username": "buivan",
    "email": "tai.buivan.jp@gmail.com",
    "isverified": false,
    "createdat": "2026-02-21T22:57:08Z"
  }
}
```

**Side effects:**
- `users.account` row created with `role = 'member'`, `isverified = FALSE`
- Verification email sent (token valid 24 hours)

**Errors:**
```json
// 409 DUPLICATE_EMAIL
{ "error": "Email already registered", "code": "DUPLICATE_EMAIL" }

// 409 DUPLICATE_USERNAME
{ "error": "Username already taken", "code": "DUPLICATE_USERNAME" }

// 400 VALIDATION_ERROR
{ "error": "Username must be 3–64 characters", "code": "VALIDATION_ERROR" }
```

---

### POST /auth/login

Authenticate and receive tokens.

**Auth required:** No  
**Rate limit:** 10 requests/min per IP (lockout after 5 failed attempts in 15 min)

**Request body:**
```json
{
  "email": "tai.buivan.jp@gmail.com",
  "password": "SuperSecret123!"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `email` | string | Yes | |
| `password` | string | Yes | |

**Response `200 OK`:**
```json
{
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "01952fa3-3f1e-7abc-b12e-1234567890ab",
      "username": "buivan",
      "email": "tai.buivan.jp@gmail.com",
      "displayname": "Bùi Văn Tài",
      "avatarurl": "https://cdn.yomira.app/avatars/buivan.webp",
      "role": "member",
      "isverified": true
    }
  }
}
```

**Response headers:**
```
Set-Cookie: refresh_token=<opaque_token>; HttpOnly; SameSite=Strict; Path=/api/v1/auth; Max-Age=2592000
```

**JWT Claims** (decoded `access_token`):
```json
{
  "sub": "01952fa3-3f1e-7abc-b12e-1234567890ab",
  "username": "buivan",
  "role": "member",
  "isverified": true,
  "iat": 1740178628,
  "exp": 1740179528
}
```

**Side effects:**
- `users.account.lastloginat` updated
- `users.session` row created (stores `SHA256(refresh_token)`, device info)

**Errors:**
```json
// 400 INVALID_CREDENTIALS
{ "error": "Invalid email or password", "code": "INVALID_CREDENTIALS" }

// 403 ACCOUNT_SUSPENDED
{ "error": "Account suspended", "code": "ACCOUNT_SUSPENDED" }

// 403 ACCOUNT_DELETED
{ "error": "Account not found", "code": "NOT_FOUND" }
```

---

### POST /auth/logout

Revoke the current session.

**Auth required:** Yes  
**Rate limit:** 30 requests/min

**Request body:** None

**Response `204 No Content`**

**Response headers:**
```
Set-Cookie: refresh_token=; HttpOnly; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```

**Side effects:**
- `users.session.revokedat` set to `NOW()` for the current session

---

### POST /auth/refresh

Exchange a valid refresh token for a new access token. Implements **refresh token rotation** — each use invalidates the old refresh token and issues a new one.

**Auth required:** No (reads `refresh_token` cookie)  
**Rate limit:** 30 requests/min per IP

**Request body:** None (reads `refresh_token` from HttpOnly cookie)

**Response `200 OK`:**
```json
{
  "data": {
    "access_token": "eyJhbGci...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Response headers:**
```
Set-Cookie: refresh_token=<new_token>; HttpOnly; SameSite=Strict; Path=/api/v1/auth; Max-Age=2592000
```

**Side effects:**
- Old `users.session` row: `revokedat = NOW()`
- New `users.session` row created with fresh token hash and `expiresat`

**Errors:**
```json
// 401 UNAUTHORIZED — missing, expired, or revoked refresh token
{ "error": "Invalid or expired refresh token", "code": "UNAUTHORIZED" }
```

---

### POST /auth/verify-email

Confirm email address with the token sent on registration.

**Auth required:** No  
**Rate limit:** 10 requests/min per IP

**Request body:**
```json
{
  "token": "f8a3b2c1d4e5..."
}
```

**Response `200 OK`:**
```json
{
  "data": { "message": "Email verified successfully" }
}
```

**Side effects:**
- `users.account.isverified = TRUE`

**Errors:**
```json
// 400 VALIDATION_ERROR — token missing or malformed
{ "error": "Invalid verification token", "code": "VALIDATION_ERROR" }
```

---

### POST /auth/forgot-password

Request a password reset email.

**Auth required:** No  
**Rate limit:** 3 requests/min per email address

**Request body:**
```json
{
  "email": "tai.buivan.jp@gmail.com"
}
```

**Response `200 OK`** (always, even if email not found — prevents user enumeration):
```json
{
  "data": { "message": "If this email is registered, a reset link has been sent." }
}
```

**Side effects:**
- If email found: sends reset email with a time-limited token (valid 1 hour)
- Token stored in Redis (not in DB) with `TTL = 3600s`

---

### POST /auth/reset-password

Set a new password using the reset token.

**Auth required:** No  
**Rate limit:** 10 requests/min per IP

**Request body:**
```json
{
  "token": "f8a3b2c1d4e5...",
  "password": "NewPassword456!"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `token` | string | Yes | Reset token from email |
| `password` | string | Yes | Min 8 chars, 1 upper, 1 lower, 1 digit. Max 72 chars. |

**Response `200 OK`:**
```json
{
  "data": { "message": "Password updated successfully" }
}
```

**Side effects:**
- `users.account.passwordhash` updated (new bcrypt hash)
- All active sessions revoked: `users.session.revokedat = NOW()` for all user sessions
- Reset token deleted from Redis

**Errors:**
```json
// 400 VALIDATION_ERROR — token expired or invalid
{ "error": "Reset token is invalid or expired", "code": "VALIDATION_ERROR" }

// 400 WEAK_PASSWORD
{ "error": "Password must be at least 8 characters with upper, lower, and digit", "code": "WEAK_PASSWORD" }
```

---

### POST /auth/change-password

Change password when already logged in.

**Auth required:** Yes  
**Rate limit:** 5 requests/min

**Request body:**
```json
{
  "current_password": "OldPassword123!",
  "new_password": "NewPassword456!"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `current_password` | string | Yes | Must match current `passwordhash` |
| `new_password` | string | Yes | Min 8 chars, 1 upper, 1 lower, 1 digit. Max 72 chars. Must differ from current. |

**Response `200 OK`:**
```json
{
  "data": { "message": "Password changed successfully" }
}
```

**Side effects:**
- `users.account.passwordhash` updated
- All **other** sessions revoked (current session stays active)

**Errors:**
```json
// 400 INVALID_CREDENTIALS — wrong current password
{ "error": "Current password is incorrect", "code": "INVALID_CREDENTIALS" }

// 400 WEAK_PASSWORD
{ "error": "New password does not meet requirements", "code": "WEAK_PASSWORD" }
```

---

## 2. Auth — OAuth

### GET /auth/oauth/:provider

Initiate OAuth2 authorization code flow. Redirects to provider's consent page.

**Auth required:** No  
**Path params:**

| Param | Values |
|---|---|
| `provider` | `google` \| `discord` \| `github` \| `apple` |

**Query params:**

| Param | Required | Description |
|---|---|---|
| `action` | No | `login` (default) or `link` (link to existing account; requires auth) |

**Response `302 Redirect`** → Provider OAuth URL

---

### GET /auth/oauth/:provider/callback

OAuth2 callback. Called by the provider after user consents. Should be registered as the redirect URI.

**Auth required:** No (state param ties to session)

**Side effects (new user):**
- `users.account` row created (`role = 'member'`, `isverified = TRUE` — email pre-verified)
- `users.oauthprovider` row created

**Side effects (existing user — same email):**
- `users.oauthprovider` row created and linked to existing account

**Response `302 Redirect`** → `/` with access token + refresh token cookie set

---

### DELETE /auth/oauth/:provider

Unlink an OAuth provider from the account.

**Auth required:** Yes  

**Business rule:** Cannot unlink if it's the only auth method and `passwordhash IS NULL` — user would be locked out.

**Response `204 No Content`**

**Side effects:**
- `users.oauthprovider` row deleted (hard delete — not soft)

**Errors:**
```json
// 409 CONFLICT — only auth method remaining
{ "error": "Cannot unlink: this is your only sign-in method. Set a password first.", "code": "CONFLICT" }

// 404 NOT_FOUND — provider not linked
{ "error": "Provider not linked to this account", "code": "NOT_FOUND" }
```

---

### GET /auth/oauth/linked

List all OAuth providers linked to the current account.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": [
    {
      "provider": "google",
      "email": "tai.buivan.jp@gmail.com",
      "linkedat": "2026-02-21T22:57:08Z"
    },
    {
      "provider": "discord",
      "email": "buivan#1234",
      "linkedat": "2026-02-22T10:00:00Z"
    }
  ]
}
```

> **Note:** `accesstoken` is **never** returned in API responses.

---

## 3. Sessions

### GET /me/sessions

List all active (non-expired, non-revoked) sessions for the current user.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": 42,
      "devicename": "Chrome on Windows",
      "ipaddress": "1.2.3.4",
      "createdat": "2026-02-21T20:00:00Z",
      "expiresat": "2026-03-22T20:00:00Z",
      "iscurrent": true
    },
    {
      "id": 38,
      "devicename": "iPhone 15",
      "ipaddress": "5.6.7.8",
      "createdat": "2026-02-18T12:00:00Z",
      "expiresat": "2026-03-18T12:00:00Z",
      "iscurrent": false
    }
  ]
}
```

> SQL: `SELECT ... FROM users.session WHERE userid = $me AND revokedat IS NULL AND expiresat > NOW()`

---

### DELETE /me/sessions/:id

Revoke a specific session (e.g., "Sign out from iPhone").

**Auth required:** Yes  
**Path params:** `id` — session ID (integer)

**Business rule:** A user can only revoke their own sessions.

**Response `204 No Content`**

**Side effects:**
- `users.session.revokedat = NOW()` for the specified session

**Errors:**
```json
// 404 NOT_FOUND
{ "error": "Session not found", "code": "NOT_FOUND" }
```

---

### DELETE /me/sessions

Revoke **all** sessions except the current one ("Sign out everywhere else").

**Auth required:** Yes  
**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `all` | bool | `false` | If `true`, revoke ALL sessions including current |

**Response `200 OK`:**
```json
{
  "data": { "revoked": 3 }
}
```

**Side effects:**
- `users.session.revokedat = NOW()` for all qualifying sessions

---

## 4. User Profile

### GET /users/:id

Get the public profile of any user.

**Auth required:** No  
**Path params:** `id` — UUIDv7

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01952fa3-3f1e-7abc-b12e-1234567890ab",
    "username": "buivan",
    "displayname": "Bùi Văn Tài",
    "avatarurl": "https://cdn.yomira.app/avatars/buivan.webp",
    "bio": "Manga enthusiast 📚",
    "website": "https://buivan.dev",
    "isverified": true,
    "followercount": 142,
    "followingcount": 38,
    "createdat": "2026-02-21T22:57:08Z",
    "isfollowing": false   // only present when caller is authenticated
  }
}
```

**Errors:**
```json
// 404
{ "error": "User not found", "code": "NOT_FOUND" }
```

---

### GET /me

Get the full private profile of the authenticated user.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": {
    "id": "01952fa3-3f1e-7abc-b12e-1234567890ab",
    "username": "buivan",
    "email": "tai.buivan.jp@gmail.com",
    "displayname": "Bùi Văn Tài",
    "avatarurl": "https://cdn.yomira.app/avatars/buivan.webp",
    "bio": "Manga enthusiast 📚",
    "website": "https://buivan.dev",
    "role": "member",
    "isverified": true,
    "isactive": true,
    "lastloginat": "2026-02-21T22:57:08Z",
    "followercount": 142,
    "followingcount": 38,
    "oauthproviders": ["google"],
    "createdat": "2026-02-21T22:57:08Z",
    "updatedat": "2026-02-21T22:57:08Z"
  }
}
```

---

### PATCH /me

Update the current user's profile. All fields optional — only provided fields are updated.

**Auth required:** Yes  
**Rate limit:** 10 requests/min

**Request body (all fields optional):**
```json
{
  "username": "buivan_new",
  "displayname": "Tai Bui Van",
  "bio": "Vietnamese manga reader",
  "website": "https://buivan.dev"
}
```

| Field | Type | Validation |
|---|---|---|
| `username` | string | 3–64 chars. `[a-zA-Z0-9_-]`. Case-insensitive unique. Cannot reclaim a taken username. |
| `displayname` | string \| null | Max 100 chars. `null` clears it. |
| `bio` | string \| null | No max defined. Reasonable limit: 500 chars enforced by Go. `null` clears it. |
| `website` | string \| null | Valid URL (must start with `https://`). `null` clears it. |

**Response `200 OK`:** Same as `GET /me`

**Errors:**
```json
// 409 DUPLICATE_USERNAME
{ "error": "Username already taken", "code": "DUPLICATE_USERNAME" }
```

---

### POST /me/avatar

Upload a new avatar image.

**Auth required:** Yes  
**Content-Type:** `multipart/form-data`  
**Rate limit:** 5 requests/min

**Request body:**
```
avatar: <file>     (JPEG/PNG/WEBP/GIF, max 5 MB)
```

| Field | Type | Validation |
|---|---|---|
| `avatar` | file | MIME: `image/jpeg`, `image/png`, `image/webp`, `image/gif`. Max 5 MB. |

**Response `200 OK`:**
```json
{
  "data": {
    "avatarurl": "https://cdn.yomira.app/avatars/01952fa3.webp"
  }
}
```

**Side effects:**
- Image processed → converted to WebP, resized to `256×256` and `64×64`
- Uploaded to object storage (R2/S3/MinIO)
- `users.account.avatarurl` updated
- `core.mediafile` row created

**Errors:**
```json
// 400 VALIDATION_ERROR — unsupported format or too large
{ "error": "Avatar must be JPEG, PNG, WEBP, or GIF and under 5 MB", "code": "VALIDATION_ERROR" }
```

---

### DELETE /me

Soft-delete the current account.

**Auth required:** Yes  
**Rate limit:** 1 request/5 min

**Request body:**
```json
{
  "password": "SuperSecret123!",
  "confirmation": "DELETE MY ACCOUNT"
}
```

| Field | Required | Notes |
|---|---|---|
| `password` | Yes (if `passwordhash` not NULL) | Verifies intent |
| `confirmation` | Yes | Must be exactly the string `"DELETE MY ACCOUNT"` |

**Response `204 No Content`**

**Side effects:**
- `users.account.deletedat = NOW()`
- All sessions revoked: `users.session.revokedat = NOW()`
- Refresh token cookie cleared

**Errors:**
```json
// 400 INVALID_CREDENTIALS
{ "error": "Incorrect password", "code": "INVALID_CREDENTIALS" }

// 400 VALIDATION_ERROR
{ "error": "Please type DELETE MY ACCOUNT to confirm", "code": "VALIDATION_ERROR" }
```

---

## 5. Follow Graph

### POST /users/:id/follow

Follow a user.

**Auth required:** Yes  
**Path params:** `id` — target user's UUIDv7

**Business rules:**
- Cannot follow yourself (`followerid ≠ followingid`)
- Cannot follow a deleted/suspended user

**Response `201 Created`:**
```json
{
  "data": {
    "followerid": "01952fa3-...",
    "followingid": "01952fa4-...",
    "createdat": "2026-02-21T23:00:00Z"
  }
}
```

**Side effects:**
- `users.follow` row inserted
- `social.notification` created for the followed user (type: `follow`)
- `social.feedevent` created (type: `user_followed`)

**Errors:**
```json
// 409 ALREADY_FOLLOWING
{ "error": "Already following this user", "code": "ALREADY_FOLLOWING" }

// 400 VALIDATION_ERROR — self-follow
{ "error": "Cannot follow yourself", "code": "VALIDATION_ERROR" }

// 404 NOT_FOUND
{ "error": "User not found", "code": "NOT_FOUND" }
```

---

### DELETE /users/:id/follow

Unfollow a user.

**Auth required:** Yes  
**Path params:** `id` — target user's UUIDv7

**Response `204 No Content`**

**Side effects:**
- `users.follow` row deleted (hard delete — junction table)

**Errors:**
```json
// 404 NOT_FOUND — not following this user
{ "error": "Not following this user", "code": "NOT_FOUND" }
```

---

### GET /users/:id/followers

List followers of a user (who follows this user).

**Auth required:** No  
**Path params:** `id` — UUIDv7  
**Query params:** `page`, `limit` (see [pagination](#pagination-query-parameters))

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-...",
      "username": "reader99",
      "displayname": null,
      "avatarurl": null,
      "isverified": false,
      "isfollowing": false,
      "followedat": "2026-02-20T10:00:00Z"
    }
  ],
  "meta": { "total": 142, "page": 1, "limit": 20, "pages": 8 }
}
```

> SQL: `SELECT a.* FROM users.follow f JOIN users.account a ON a.id = f.followerid WHERE f.followingid = $id ORDER BY f.createdat DESC`

---

### GET /users/:id/following

List users that this user follows (who does this user follow).

**Auth required:** No  
**Path params:** `id` — UUIDv7  
**Query params:** `page`, `limit`

**Response `200 OK`:** Same shape as `/followers` but from the `followingid` side.

---

### GET /me/followers

Shorthand for `GET /users/{my_id}/followers` — list my followers.

**Auth required:** Yes

**Response:** Same as `/users/:id/followers`

---

### GET /me/following

Shorthand for `GET /users/{my_id}/following` — list who I follow.

**Auth required:** Yes

**Response:** Same as `/users/:id/following`

---

## 6. Reading Preferences

### GET /me/preferences

Get current reader UI preferences. Returns defaults if not yet saved.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": {
    "readingmode": "ltr",
    "pagefit": "width",
    "doublepageon": false,
    "showpagebar": true,
    "preloadpages": 3,
    "datasaver": false,
    "hidensfw": true,
    "hidelanguages": [],
    "updatedat": "2026-02-21T22:57:08Z"
  }
}
```

> If no row in `users.readingpreference`, return the column defaults (do not 404).

---

### PUT /me/preferences

Save reader preferences. This is an **upsert** — creates the row if it doesn't exist, replaces it if it does. All fields required (full replacement).

**Auth required:** Yes

**Request body:**
```json
{
  "readingmode": "webtoon",
  "pagefit": "width",
  "doublepageon": false,
  "showpagebar": true,
  "preloadpages": 5,
  "datasaver": true,
  "hidensfw": false,
  "hidelanguages": ["ja", "zh-hans"]
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `readingmode` | string | Yes | `ltr` \| `rtl` \| `vertical` \| `webtoon` |
| `pagefit` | string | Yes | `width` \| `height` \| `original` \| `stretch` |
| `doublepageon` | bool | Yes | |
| `showpagebar` | bool | Yes | |
| `preloadpages` | int | Yes | `1`–`10` |
| `datasaver` | bool | Yes | |
| `hidensfw` | bool | Yes | |
| `hidelanguages` | string[] | Yes | Valid BCP-47 codes. Max 50 entries. |

**Response `200 OK`:** Same as `GET /me/preferences`

**Side effects:**
- `INSERT INTO users.readingpreference ... ON CONFLICT (userid) DO UPDATE SET ...`

**Errors:**
```json
// 400 VALIDATION_ERROR
{ "error": "readingmode must be one of: ltr, rtl, vertical, webtoon", "code": "VALIDATION_ERROR" }

// 400 VALIDATION_ERROR
{ "error": "preloadpages must be between 1 and 10", "code": "VALIDATION_ERROR" }
```

---

## 7. Admin — User Management

> All endpoints in this section require `role = 'admin'`.

### GET /admin/users

List all users with filtering and full data.

**Auth required:** Yes (admin)

**Query params:**

| Param | Type | Description |
|---|---|---|
| `page` | int | Page number |
| `limit` | int | Items per page (max 100) |
| `role` | string | Filter by role: `admin` \| `moderator` \| `member` \| `banned` |
| `isverified` | bool | Filter by verification status |
| `isactive` | bool | Filter by active status |
| `deleted` | bool | If `true`, include soft-deleted accounts |
| `q` | string | Search by username or email (trigram) |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "01952fa3-...",
      "username": "buivan",
      "email": "tai.buivan.jp@gmail.com",
      "role": "member",
      "isverified": true,
      "isactive": true,
      "lastloginat": "2026-02-21T22:57:08Z",
      "createdat": "2026-02-21T22:57:08Z",
      "deletedat": null
    }
  ],
  "meta": { "total": 5432, "page": 1, "limit": 20, "pages": 272 }
}
```

---

### GET /admin/users/:id

Get full account details for any user (admin view).

**Auth required:** Yes (admin)

**Response `200 OK`:** Full `UserPrivate` + `deletedat` field.

---

### PATCH /admin/users/:id/role

Change a user's role.

**Auth required:** Yes (admin)

**Request body:**
```json
{
  "role": "moderator",
  "reason": "Promoted to moderator — active community member"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `role` | string | Yes | `admin` \| `moderator` \| `member` \| `banned` |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:** Updated user object.

**Side effects:**
- `users.account.role` updated
- `system.auditlog` row created (action: `user.role_change`)

---

### PATCH /admin/users/:id/suspend

Suspend or un-suspend an account.

**Auth required:** Yes (admin)

**Request body:**
```json
{
  "suspend": true,
  "reason": "Violated community guidelines — spam"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `suspend` | bool | Yes | `true` = suspend (`isactive = FALSE`), `false` = restore |
| `reason` | string | No | Stored in `system.auditlog` |

**Response `200 OK`:** Updated user object.

**Side effects:**
- `users.account.isactive` updated
- If `suspend = true`: all sessions revoked
- `system.auditlog` row created (action: `user.suspend` or `user.restore`)

---

### DELETE /admin/users/:id

Soft-delete a user account (admin-initiated).

**Auth required:** Yes (admin)

**Request body:**
```json
{
  "reason": "Permanent ban — CSAM violation"
}
```

**Response `204 No Content`**

**Side effects:**
- `users.account.deletedat = NOW()`
- All sessions revoked
- `system.auditlog` row created (action: `user.delete`)

---

## Implementation Notes for Go Developers

### Validation checklist (Go service layer — NOT SQL)

```go
// users.account
username: len 3–64, regex [a-zA-Z0-9_-], case-insensitive unique check
email:    valid format, max 254 chars, unique check
password: len 8–72, 1 upper + 1 lower + 1 digit minimum
role:     domain.UserRole.IsValid()  → "admin"|"moderator"|"member"|"banned"

// users.readingpreference
readingmode:    domain.ReadingMode.IsValid()   → "ltr"|"rtl"|"vertical"|"webtoon"
pagefit:        domain.PageFit.IsValid()       → "width"|"height"|"original"|"stretch"
preloadpages:   1 <= n <= 10
hidelanguages:  each code must be valid BCP-47 (check against core.language.code)

// users.follow
followerid != followingid   (self-follow prevention)
```

### Auth middleware context key

```go
// internal/auth/context.go
type contextKey string
const userKey contextKey = "auth_user"

type AuthUser struct {
    ID         string
    Username   string
    Role       domain.UserRole
    IsVerified bool
}

func UserFromCtx(ctx context.Context) (*AuthUser, bool) {
    u, ok := ctx.Value(userKey).(*AuthUser)
    return u, ok
}
```

### Session creation on login

```go
// Pseudocode — internal/auth/service.go
refreshToken := crypto.RandomBytes(32)      // 32 bytes = 256-bit entropy
tokenHash   := sha256.Sum256(refreshToken)
session := &domain.Session{
    UserID:     user.ID,
    TokenHash:  hex.EncodeToString(tokenHash[:]),
    DeviceName: parseUserAgent(r.Header.Get("User-Agent")),
    IPAddress:  extractIP(r),
    ExpiresAt:  time.Now().Add(30 * 24 * time.Hour),
}
sessionRepo.Create(ctx, session)
// Set raw refreshToken in HttpOnly cookie
// Never store raw token in DB
```
