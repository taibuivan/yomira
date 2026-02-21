# API Reference — Mail / Transactional Email

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Base URL:** `/api/v1`  
> **Content-Type:** `application/json`

> **Architecture note:** Email is a **pure Go service** — no dedicated database schema.  
> Transactional emails are sent via an external SMTP provider (e.g. Resend, SendGrid, AWS SES, Postmark).  
> **Tokens** (verification, password reset, email change) are **short-lived JWTs** or **signed HMAC URLs** — never stored in the database.  
> Rate limits are enforced via **Redis** (per IP and per email address).

---

## Changelog

| Version | Date | Changes |
|---|---|---|
| **1.0.0** | 2026-02-22 | Initial release. Email verification, password reset, email change, digest, admin preview. |

---

## Table of Contents

1. [Email Flow Overview](#1-email-flow-overview)
2. [Email Verification](#2-email-verification)
3. [Password Reset](#3-password-reset)
4. [Email Address Change](#4-email-address-change)
5. [Notification Digest](#5-notification-digest)
6. [Admin — Mail Management](#6-admin--mail-management)
7. [Internal Mail Events (Go Service Only)](#7-internal-mail-events-go-service-only)
8. [Implementation Notes](#8-implementation-notes)

---

## Endpoint Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/email/verify/send` | No* | Resend verification email |
| `GET` | `/auth/email/verify/:token` | No | Confirm email with token from email link |
| `POST` | `/auth/password/forgot` | No | Request password reset email |
| `POST` | `/auth/password/reset` | No | Reset password using token |
| `POST` | `/me/email/change/request` | Yes | Request email address change (sends verify to new address) |
| `GET` | `/me/email/change/confirm/:token` | No | Confirm new email address with token |
| `DELETE` | `/me/email/change/cancel` | Yes | Cancel pending email change |
| `PATCH` | `/me/notifications/email` | Yes | Update email notification preferences |
| `GET` | `/me/notifications/email` | Yes | Get current email notification preferences |
| `POST` | `/me/notifications/unsubscribe` | No | Unsubscribe from email notifications (via link in email) |
| `GET` | `/admin/mail/templates` | admin | List available email templates |
| `GET` | `/admin/mail/templates/:slug` | admin | Get an email template |
| `PATCH` | `/admin/mail/templates/:slug` | admin | Update an email template |
| `POST` | `/admin/mail/templates/:slug/preview` | admin | Preview rendered email (no send) |
| `POST` | `/admin/mail/send` | admin | Send a custom email to one or more users |
| `GET` | `/admin/mail/logs` | admin | View sent email logs |

---

## 1. Email Flow Overview

```
User action                       Go service                     SMTP provider
─────────────────────────────────────────────────────────────────────────────
POST /auth/register           →   Create account (isverified=false)
                              →   Generate signed token (JWT, 24h TTL)
                              →   Queue email: "verify-email" template
                              →   mailService.Send(to, template, data)  ──→  SMTP → User inbox

GET /auth/email/verify/:token →   Validate token signature + expiry
                              →   SET users.account.isverified = TRUE
                              →   Redirect to /welcome

POST /auth/password/forgot    →   Look up email
                              →   Generate signed reset token (JWT, 1h TTL)
                              →   Queue email: "password-reset" template   ──→  SMTP → User inbox

POST /auth/password/reset     →   Validate reset token
                              →   Hash new password, UPDATE passwordhash
                              →   Revoke all sessions (security)
                              →   Send confirmation email: "password-changed"
```

---

## 2. Email Verification

### POST /auth/email/verify/send

Resend the verification email to the currently logged-in user (or by email address).

**Auth required:** No (can be called before login with just email)

**Rate limit:** 3 requests per email per 15 minutes (Redis key: `ratelimit:email-verify:{email}`)

**Request body:**
```json
{ "email": "user@example.com" }
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `email` | string | Yes | Must match a registered `users.account.email` that is `isverified = FALSE` |

**Response `204 No Content`**  
(Response is always 204 regardless of whether the email exists — prevents user enumeration.)

**Side effects:**
- Signed verification token generated (JWT, 24h TTL, signed with `JWT_PRIVATE_KEY`)
- Email queued: template `verify-email`, sent to the address

**Errors:**
```json
{ "error": "Too many requests. Please wait before requesting another verification email.", "code": "RATE_LIMITED" }
```

---

### GET /auth/email/verify/:token

Confirm email address using the token from the verification link.

**Auth required:** No  
**Path params:** `token` — signed JWT from the email link

**Response `302 Found`** — redirects to:
- **Success:** `/welcome?verified=true`
- **Expired:** `/auth/verify-expired`
- **Already verified:** `/auth/login?msg=already_verified`

**Side effects (on success):**
- `users.account.isverified = TRUE`
- Welcome notification created (`social.notification`, type: `system`)
- Confirmation email sent: template `email-verified`

**Errors (redirect-based):**

| Scenario | Redirect |
|---|---|
| Token expired (> 24h) | `/auth/verify-expired?email={encoded}` |
| Token invalid / tampered | `/auth/error?code=invalid_token` |
| Account not found | `/auth/error?code=account_not_found` |

---

## 3. Password Reset

### POST /auth/password/forgot

Request a password reset email.

**Auth required:** No  
**Rate limit:** 3 requests per email per 1 hour (Redis key: `ratelimit:password-reset:{email}`)

**Request body:**
```json
{ "email": "user@example.com" }
```

**Response `204 No Content`**  
(Always 204 — prevents email enumeration even if account does not exist.)

**Side effects (if email matches an account):**
- Signed reset token generated (JWT, **1 hour TTL**, signed with `JWT_PRIVATE_KEY`)
- Token payload: `{ sub: userID, email: email, type: "password_reset", iat: ..., exp: ... }`
- Email queued: template `password-reset`

**Errors:**
```json
{ "error": "Too many requests. Please wait before requesting another reset email.", "code": "RATE_LIMITED" }
```

---

### POST /auth/password/reset

Reset password using the token from the reset email.

**Auth required:** No

**Request body:**
```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "password": "NewSecureP@ssw0rd!",
  "passwordconfirm": "NewSecureP@ssw0rd!"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `token` | string | Yes | Signed JWT from reset email |
| `password` | string | Yes | Min 8 chars, max 128 chars. Must contain uppercase, lowercase, digit. |
| `passwordconfirm` | string | Yes | Must match `password` |

**Response `200 OK`:**
```json
{ "data": { "message": "Password reset successfully. Please log in with your new password." } }
```

**Side effects:**
- Token validated (signature + expiry + `type = "password_reset"`)
- `users.account.passwordhash` updated with bcrypt hash (cost=12)
- **All sessions revoked:** `UPDATE users.session SET revokedat = NOW() WHERE userid = $1 AND revokedat IS NULL`
- Email sent: template `password-changed` (security notice)
- `system.auditlog` written (action: `user.password_reset`)

**Errors:**
```json
{ "error": "Reset token has expired. Please request a new password reset.", "code": "TOKEN_EXPIRED" }
{ "error": "Invalid reset token.", "code": "TOKEN_INVALID" }
{ "error": "Passwords do not match.", "code": "VALIDATION_ERROR" }
{ "error": "Password must be at least 8 characters.", "code": "VALIDATION_ERROR" }
```

---

## 4. Email Address Change

### POST /me/email/change/request

Request to change the account's email address. Sends a verification email to the **new** address.

**Auth required:** Yes

**Request body:**
```json
{
  "newemail": "newaddress@example.com",
  "password": "CurrentP@ssw0rd!"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `newemail` | string | Yes | Valid email format. Max 254 chars. Must not already be in use. |
| `password` | string | Yes | Current password (required to confirm ownership). Not required for OAuth-only accounts — use `currentpassword: null`. |

**Response `204 No Content`**

**Side effects:**
- Current password verified (bcrypt compare)
- Signed change token generated (JWT, **1 hour TTL**)
- Token payload: `{ sub: userID, oldemail: "...", newemail: "...", type: "email_change" }`
- Email queued to `newemail`: template `email-change-confirm`
- Email sent to `oldemail`: template `email-change-notice` (security alert — "someone requested an email change")

**Rate limit:** 3 requests per user per 24 hours (Redis key: `ratelimit:email-change:{userid}`)

**Errors:**
```json
{ "error": "This email address is already in use.", "code": "CONFLICT" }
{ "error": "Current password is incorrect.", "code": "FORBIDDEN" }
{ "error": "Too many email change requests. Please try again tomorrow.", "code": "RATE_LIMITED" }
```

---

### GET /me/email/change/confirm/:token

Confirm new email address using the token from the email link.

**Auth required:** No (token is self-authenticating)  
**Path params:** `token` — signed JWT from the confirmation email

**Response `302 Found`** — redirects to:
- **Success:** `/settings/profile?msg=email_changed`
- **Expired:** `/settings/profile?error=token_expired`

**Side effects (on success):**
- Token validated (signature + expiry + `type = "email_change"`)
- `users.account.email = newemail` updated
- All sessions revoked (security: new email = new identity)
- Email sent to `oldemail`: template `email-changed-notice` (confirmation)
- `system.auditlog` written (action: `user.email_change`)

---

### DELETE /me/email/change/cancel

Cancel a pending email change request (invalidates the change token).

**Auth required:** Yes

**Response `204 No Content`**

**Side effects:**
- Redis key `email_change_token:{userid}` deleted (token blacklisted)
- Email sent to current address: template `email-change-cancelled`

---

## 5. Notification Digest

### GET /me/notifications/email

Get the current user's email notification preferences.

**Auth required:** Yes

**Response `200 OK`:**
```json
{
  "data": {
    "enabled": true,
    "preferences": {
      "new_chapter": true,
      "comment_reply": true,
      "follow": false,
      "announcement": true,
      "digest_weekly": false
    },
    "unsubscribetoken": null
  }
}
```

> Preferences stored in `users.readingpreference.emailprefs` (JSONB column — added on migration).

---

### PATCH /me/notifications/email

Update email notification preferences.

**Auth required:** Yes

**Request body:**
```json
{
  "enabled": true,
  "preferences": {
    "new_chapter": true,
    "comment_reply": false,
    "follow": false,
    "announcement": true,
    "digest_weekly": false
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `enabled` | bool | No | Master switch — `false` unsubscribes from all emails |
| `preferences` | object | No | Individual event toggles. Keys: `new_chapter` \| `comment_reply` \| `follow` \| `announcement` \| `digest_weekly` |

**Response `200 OK`:** Updated preferences object.

**Side effects:** `users.readingpreference.emailprefs` updated (upsert).

---

### POST /me/notifications/unsubscribe

One-click unsubscribe — used from the unsubscribe link in email footers. No login required.

**Auth required:** No

**Query params:**

| Param | Type | Required | Notes |
|---|---|---|---|
| `token` | string | Yes | Signed HMAC unsubscribe token from email footer (encodes `userid` + `type`) |
| `type` | string | No | `all` (default) \| `new_chapter` \| `comment_reply` \| `announcement` — which email type to unsubscribe from |

**Request body:** None

**Response `200 OK`:**
```json
{ "data": { "message": "You have been unsubscribed from new_chapter emails." } }
```

**Side effects:**
- `users.readingpreference.emailprefs.{type} = false` (or `enabled = false` for `all`)
- Token is single-use — blacklisted in Redis (`SET unsubscribe:{token} 1 EX 86400`)

**Errors:**
```json
{ "error": "Invalid or expired unsubscribe token.", "code": "TOKEN_INVALID" }
```

---

## 6. Admin — Mail Management

### GET /admin/mail/templates

List all available email templates.

**Auth required:** Yes (role: `admin`)

**Response `200 OK`:**
```json
{
  "data": [
    { "slug": "verify-email",         "subject": "Verify your Yomira email address",    "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "password-reset",       "subject": "Reset your Yomira password",           "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "password-changed",     "subject": "Your Yomira password was changed",     "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "email-change-confirm", "subject": "Confirm your new Yomira email address","updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "email-change-notice",  "subject": "Email change requested on your Yomira account","updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "email-changed-notice", "subject": "Your Yomira email address was changed","updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "email-verified",       "subject": "Welcome to Yomira!",                   "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "new-chapter",          "subject": "{{comic}} — Chapter {{number}} is out!","updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "comment-reply",        "subject": "{{username}} replied to your comment", "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "announcement",         "subject": "{{title}}",                            "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "weekly-digest",        "subject": "Your Yomira Weekly — {{date}}",        "updatedAt": "2026-02-01T00:00:00Z" },
    { "slug": "account-deleted",      "subject": "Your Yomira account has been deleted", "updatedAt": "2026-02-01T00:00:00Z" }
  ]
}
```

---

### GET /admin/mail/templates/:slug

Get a single email template with its full HTML/text body.

**Auth required:** Yes (role: `admin`)  
**Path params:** `slug` — template slug

**Response `200 OK`:**
```json
{
  "data": {
    "slug": "password-reset",
    "subject": "Reset your Yomira password",
    "bodyhtml": "<html>...<a href=\"{{reseturl}}\">Reset Password</a>...</html>",
    "bodytext": "Reset your password: {{reseturl}}",
    "variables": ["reseturl", "username", "expiresin"],
    "updatedAt": "2026-02-01T00:00:00Z"
  }
}
```

---

### PATCH /admin/mail/templates/:slug

Update an email template (subject, HTML body, or text body).

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{
  "subject": "Reset your Yomira password - Action Required",
  "bodyhtml": "<html>...<a href=\"{{reseturl}}\">Reset Password</a>...</html>",
  "bodytext": "Reset your password here: {{reseturl}}\n\nThis link expires in {{expiresin}}."
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `subject` | string | No | Supports `{{variable}}` interpolation |
| `bodyhtml` | string | No | Full HTML. Supports `{{variable}}` interpolation. |
| `bodytext` | string | No | Plaintext fallback. |

**Response `200 OK`:** Updated template object.

**Side effects:** Template stored (file system or DB-backed store). `system.auditlog` written.

---

### POST /admin/mail/templates/:slug/preview

Render an email template with sample data — **no email is sent**.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{
  "variables": {
    "username": "buivan",
    "reseturl": "https://yomira.app/auth/reset?token=example",
    "expiresin": "1 hour"
  },
  "to": "preview@example.com"
}
```

**Response `200 OK`:**
```json
{
  "data": {
    "slug": "password-reset",
    "subject": "Reset your Yomira password",
    "bodyhtml": "<html>...<a href=\"https://yomira.app/auth/reset?token=example\">Reset Password</a>...</html>",
    "bodytext": "Reset your password here: https://yomira.app/auth/reset?token=example\n\nThis link expires in 1 hour."
  }
}
```

---

### POST /admin/mail/send

Send a custom transactional email to one or more users. Used for announcements or support.

**Auth required:** Yes (role: `admin`)

**Request body:**
```json
{
  "to": ["user@example.com", "01952fa3-..."],
  "subject": "Important update to your account",
  "bodyhtml": "<p>Dear user, ...</p>",
  "bodytext": "Dear user, ...",
  "template": null
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `to` | string[] | Yes | Email addresses or user UUIDs. Max 100 recipients per call. |
| `subject` | string | Yes* | Required if `template` is null. Max 300 chars. |
| `bodyhtml` | string | No | HTML body. Required if `template` is null and `bodytext` is null. |
| `bodytext` | string | No | Plaintext fallback. |
| `template` | string \| null | No | If provided, uses an existing template slug. `subject`, `bodyhtml`, `bodytext` are ignored. |
| `variables` | object | No | Template variable values (if `template` is set). |

**Response `200 OK`:**
```json
{ "data": { "queued": 3, "failed": 0 } }
```

**Side effects:** Emails queued in SMTP provider. `system.auditlog` written (action: `mail.admin_send`).

**Errors:**
```json
{ "error": "Maximum 100 recipients per send", "code": "VALIDATION_ERROR" }
{ "error": "User not found: 01952fa3-...", "code": "NOT_FOUND" }
```

---

### GET /admin/mail/logs

View sent email logs from the SMTP provider (proxied from provider API).

**Auth required:** Yes (role: `admin`)

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `to` | string | — | Filter by recipient email |
| `template` | string | — | Filter by template slug |
| `status` | string | — | `sent` \| `delivered` \| `bounced` \| `complained` \| `failed` |
| `from` | string | 7 days ago | ISO 8601 |
| `to_date` | string | `NOW()` | ISO 8601 |
| `page` | int | `1` | — |
| `limit` | int | `50` | Max `200` |

**Response `200 OK`:**
```json
{
  "data": [
    {
      "id": "provider-msg-id-001",
      "to": "user@example.com",
      "subject": "Reset your Yomira password",
      "template": "password-reset",
      "status": "delivered",
      "sentAt": "2026-02-22T00:00:00Z",
      "deliveredAt": "2026-02-22T00:00:02Z",
      "openedAt": "2026-02-22T00:01:30Z"
    }
  ],
  "meta": { "total": 842, "page": 1, "limit": 50, "pages": 17 }
}
```

> Email logs are **fetched from the SMTP provider API** (e.g. Resend Events API, SendGrid Activity API), not stored locally.

---

## 7. Internal Mail Events (Go Service Only)

> These are **not HTTP endpoints** — they are `mailService.Send(...)` calls made automatically inside Go as side-effects of other operations.

| Trigger | Template | Recipient |
|---|---|---|
| `POST /auth/register` | `verify-email` | new user |
| `GET /auth/email/verify/:token` ✅ | `email-verified` | user |
| `POST /auth/password/forgot` | `password-reset` | user |
| `POST /auth/password/reset` ✅ | `password-changed` | user |
| `POST /me/email/change/request` | `email-change-confirm` | new email |
| `POST /me/email/change/request` | `email-change-notice` | old email (security alert) |
| `GET /me/email/change/confirm/:token` ✅ | `email-changed-notice` | old email |
| `DELETE /me/email/change/cancel` | `email-change-cancelled` | current email |
| `DELETE /admin/users/:id` | `account-deleted` | deleted user |
| `PATCH /admin/users/:id/suspend` | `account-suspended` | suspended user |
| New chapter on followed comic | `new-chapter` | follower (if `emailprefs.new_chapter = true`) |
| New comment reply | `comment-reply` | parent comment author (if `emailprefs.comment_reply = true`) |
| Announcement published | `announcement` | all users (if `emailprefs.announcement = true`) |
| Weekly digest job | `weekly-digest` | users (if `emailprefs.digest_weekly = true`) |

---

## 8. Implementation Notes

### Token design (no DB storage)

Email tokens are **signed JWTs** (RS256, same key pair as auth) with a `type` claim:

```go
// Generate verification token
token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
    "sub":  userID,
    "email": email,
    "type": "email_verify",   // "password_reset" | "email_change" | etc.
    "iat":  time.Now().Unix(),
    "exp":  time.Now().Add(24 * time.Hour).Unix(),
})
signed, _ := token.SignedString(privateKey)
```

> **Why no DB?** Avoids a `emailtoken` table; tokens are self-validating.  
> **Revocation:** For security-critical tokens (password reset), store a `jti` blacklist in Redis with TTL = token TTL.

### Unsubscribe token design

```go
// HMAC-SHA256 signed token (not JWT — simpler for one-click links)
mac := hmac.New(sha256.New, []byte(UNSUBSCRIBE_SECRET))
mac.Write([]byte(userID + ":" + emailType + ":" + timestamp))
token := base64.URLEncoding.EncodeToString(mac.Sum(nil))

// Embed in email footer:
// https://yomira.app/api/v1/me/notifications/unsubscribe?token={token}&type=new_chapter
```

### Rate limiting (Redis)

```go
// Per-email rate limit — checked before sending
key := fmt.Sprintf("ratelimit:email-verify:%s", email)
count, _ := redis.Incr(ctx, key)
if count == 1 {
    redis.Expire(ctx, key, 15*time.Minute)
}
if count > 3 {
    return ErrRateLimited
}
```

### SMTP provider integration

```go
// mailService wraps the provider SDK
type MailService interface {
    Send(ctx context.Context, msg MailMessage) error
}

// Provider implementations (swap via config):
// - ResendMailService    (default)
// - SendGridMailService
// - SESMailService
// - SMTPMailService      (generic, for local dev)

type MailMessage struct {
    To       []string
    Subject  string
    BodyHTML string
    BodyText string
    From     string   // default: "noreply@yomira.app"
    ReplyTo  string   // default: "support@yomira.app"
}
```

### Environment variables

| Variable | Example | Description |
|---|---|---|
| `MAIL_PROVIDER` | `resend` | `resend` \| `sendgrid` \| `ses` \| `smtp` |
| `MAIL_FROM` | `noreply@yomira.app` | Default sender address |
| `MAIL_REPLY_TO` | `support@yomira.app` | Reply-to address |
| `RESEND_API_KEY` | `re_...` | Resend API key |
| `SENDGRID_API_KEY` | `SG...` | SendGrid API key |
| `SMTP_HOST` | `smtp.mailhog.local` | Generic SMTP host (dev/fallback) |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASS` | — | SMTP password |
| `UNSUBSCRIBE_SECRET` | `changeme` | HMAC secret for unsubscribe tokens |
| `APP_URL` | `https://yomira.app` | Base URL used in email links |
