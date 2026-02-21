# Security Reference — Yomira Backend

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Threat Model](#1-threat-model)
2. [Authentication Security](#2-authentication-security)
3. [Authorization](#3-authorization)
4. [Input Validation & Sanitization](#4-input-validation--sanitization)
5. [SQL Injection Prevention](#5-sql-injection-prevention)
6. [Secrets Management](#6-secrets-management)
7. [Data Privacy](#7-data-privacy)
8. [HTTP Security Headers](#8-http-security-headers)
9. [Rate Limiting & Abuse Prevention](#9-rate-limiting--abuse-prevention)
10. [Dependency Security](#10-dependency-security)
11. [Security Checklist](#11-security-checklist)

---

## 1. Threat Model

### Assets to protect

| Asset | Sensitivity | Where stored |
|---|---|---|
| User passwords | Critical | bcrypt hash in `users.account.passwordhash` |
| Refresh tokens | Critical | SHA-256 hash in `users.session.tokenhash` |
| OAuth access tokens | High | Encrypted in `users.oauthprovider.accesstoken` |
| JWT private key | Critical | Env var / secret manager |
| User emails | High | Plaintext in `users.account.email` |
| User IP addresses | Medium | `analytics.pageview`, `users.session` — anonymized after 90d |
| Payment data | N/A | No payments in v1.0.0 |

### Key threats

| Threat | Mitigation |
|---|---|
| Credential stuffing | Rate limit on `/auth/login`; lockout after 5 failed attempts |
| Password brute force | bcrypt cost=12; rate limit; account lockout |
| Session hijacking | Refresh token rotation; SHA-256 storage; HttpOnly cookie |
| JWT forgery | RS256 (asymmetric) — verify with public key only |
| CSRF | SameSite=Strict cookie; Bearer token in header (not cookie) |
| XSS → token theft | Access token in memory only (not localStorage); HttpOnly refresh cookie |
| SQL injection | Parameterized queries only; no string concatenation |
| Mass assignment | Explicit field whitelisting in all request parsing |
| IDOR | Owner check on all resource mutations |
| Privilege escalation | Role hierarchy enforced in middleware, not DB |
| Enumeration attacks | Consistent 204 on `/auth/forgot-password` regardless of email exists |
| DDoS | Rate limiting per IP and per user; Redis counters |
| Crawler abuse | `robots.txt`; strict API rate limits |

---

## 2. Authentication Security

### Password hashing

```go
// bcrypt cost=12 — ~250ms on modern hardware.
// Chosen to make brute-force expensive while keeping login UX acceptable.
const bcryptCost = 12

func HashPassword(plain string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
    return string(hash), err
}

func CheckPassword(plain, hash string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
```

**Rules:**
- Max password length: **128 chars** (Go-validated, not bcrypt 72-byte truncation)
- Minimum: 8 chars, 1 uppercase, 1 lowercase, 1 digit
- Never log passwords — not even masked

### JWT — RS256 (asymmetric)

```go
// Why RS256 over HS256:
// - Public key can be shared with other services (e.g. admin service) for verification
// - Private key stays in one place (API service only)
// - Easier key rotation — update public key consumers independently

// Key generation (one-time, store securely):
// openssl genrsa -out jwt_private.pem 2048
// openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem

// JWT payload
type Claims struct {
    jwt.RegisteredClaims             // sub, iat, exp, jti
    Role       string `json:"role"`
    IsVerified bool   `json:"isverified"`
}

func IssueAccessToken(user *User, privateKey *rsa.PrivateKey) (string, error) {
    claims := Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID,
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            ID:        uuidv7.New().String(),   // jti — unique per token
        },
        Role:       user.Role,
        IsVerified: user.IsVerified,
    }
    return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey)
}
```

### Refresh token security

```go
// Raw refresh token: 32 random bytes → base64url
func GenerateRefreshToken() (raw string, hash string, err error) {
    buf := make([]byte, 32)
    if _, err = rand.Read(buf); err != nil {
        return
    }
    raw = base64.URLEncoding.EncodeToString(buf)
    // Store only SHA-256 hash — raw is sent to client once and discarded
    h := sha256.Sum256([]byte(raw))
    hash = hex.EncodeToString(h[:])
    return
}

// Set as HttpOnly cookie — inaccessible to JavaScript
func SetRefreshCookie(w http.ResponseWriter, token string, ttl time.Duration) {
    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    token,
        Path:     "/api/v1/auth",       // restrict to auth endpoints only
        MaxAge:   int(ttl.Seconds()),
        HttpOnly: true,                 // JS cannot read this
        Secure:   true,                 // HTTPS only in production
        SameSite: http.SameSiteStrictMode,  // prevents CSRF
    })
}
```

### Refresh token rotation

Every use of a refresh token **invalidates the old one** and issues a new one. This means a stolen token can only be used once — after legitimate use, the attacker's copy becomes invalid.

```go
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*Tokens, error) {
    hash := sha256Hex(rawToken)

    session, err := s.repo.FindSessionByTokenHash(ctx, hash)
    if err != nil || session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
        // Possible token reuse — revoke ALL sessions for this user (security event)
        if session != nil {
            _ = s.repo.RevokeAllUserSessions(ctx, session.UserID)
            s.notifySecurityEvent(ctx, session.UserID, "refresh_token_reuse")
        }
        return nil, ErrInvalidRefreshToken
    }

    // Rotate: revoke old, issue new
    _ = s.repo.RevokeSession(ctx, session.ID)
    return s.issueNewSession(ctx, session.UserID)
}
```

### Login brute-force protection

```go
// After each failed login attempt, increment Redis counter
const (
    loginFailMaxAttempts = 5
    loginFailWindow      = 15 * time.Minute
    loginLockDuration    = 30 * time.Minute
)

func (s *AuthService) checkLoginRateLimit(ctx context.Context, email string) error {
    key := "login:fail:" + strings.ToLower(email)
    count, _ := s.redis.Incr(ctx, key)
    if count == 1 {
        s.redis.Expire(ctx, key, loginFailWindow)
    }
    if count > loginFailMaxAttempts {
        return &apperr.AppError{
            Code:       "ACCOUNT_LOCKED",
            Message:    "Too many failed login attempts. Please try again in 30 minutes.",
            HTTPStatus: http.StatusTooManyRequests,
        }
    }
    return nil
}
```

---

## 3. Authorization

### Owner checks (IDOR prevention)

**Always** verify that the authenticated user owns or has access to the resource being mutated.

```go
// ✅ Correct — always check ownership
func (s *CommentService) Delete(ctx context.Context, commentID string) error {
    caller := auth.MustGetUser(ctx)
    comment, err := s.repo.FindByID(ctx, commentID)
    if err != nil {
        return err
    }
    // Owner or admin/mod can delete
    if comment.UserID != caller.ID && !auth.IsModOrAbove(caller.Role) {
        return apperr.Forbidden("You do not have permission to delete this comment")
    }
    return s.repo.SoftDelete(ctx, commentID)
}

// ❌ Wrong — missing ownership check
func (s *CommentService) Delete(ctx context.Context, commentID string) error {
    return s.repo.SoftDelete(ctx, commentID)  // anyone can delete any comment!
}
```

### Role guard in service layer (defense in depth)

The middleware enforces role at the route level. The service layer adds a **second check** for critical mutations:

```go
func (s *UserService) ChangeRole(ctx context.Context, targetUserID, newRole string) error {
    caller := auth.MustGetUser(ctx)
    if caller.Role != "admin" {
        return apperr.Forbidden("Only admins can change roles")  // double-check
    }
    if newRole == "admin" && caller.ID == targetUserID {
        return apperr.Forbidden("Admins cannot promote themselves")
    }
    // ... proceed
}
```

---

## 4. Input Validation & Sanitization

### Never trust client input

```go
// ✅ Whitelist allowed fields (explicit struct binding)
type UpdateComicRequest struct {
    Title       *string   `json:"title"`
    Description *string   `json:"description"`
    Status      *string   `json:"status"`
    // No "id", "createdat", "followcount" — mass assignment impossible
}

// ❌ Never use map[string]any to update DB columns
func update(data map[string]any) {
    // Attacker can inject arbitrary column names
}
```

### String sanitization rules

```go
// Applied by Go validator — NOT by DB
const (
    MaxTitleLen       = 500
    MaxBodyLen        = 100_000
    MaxBioLen         = 1000
    MaxCommentBodyLen = 10_000
    MaxUsernameLen    = 64
    MaxEmailLen       = 254
)

func SanitizeString(s string) string {
    s = strings.TrimSpace(s)              // trim whitespace
    s = strings.Map(removeBadChars, s)    // strip null bytes, control chars
    return s
}

// Username: strict allowlist
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

// Email: RFC 5322 subset
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Slug: lowercase, hyphens only
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
```

### HTML content

Forum posts and announcements support Markdown. **Never render raw HTML from user input** — always sanitize:

```go
import "github.com/microcosm-cc/bluemonday"

var policy = bluemonday.UGCPolicy()  // user-generated content policy

func SanitizeHTML(raw string) string {
    return policy.Sanitize(raw)    // strips dangerous tags (script, iframe, etc.)
}
```

---

## 5. SQL Injection Prevention

**Rule: All database queries use parameterized statements. No exceptions.**

```go
// ✅ Correct — parameterized
db.QueryRowContext(ctx,
    `SELECT id, title FROM core.comic WHERE id = $1 AND deletedat IS NULL`,
    comicID)

// ❌ Wrong — string concatenation
db.QueryRowContext(ctx,
    `SELECT id, title FROM core.comic WHERE id = '` + comicID + `'`)

// ✅ Correct — dynamic sort column (whitelist approach)
allowedSortColumns := map[string]string{
    "title":   "c.title",
    "rating":  "c.ratingbayesian",
    "follows": "c.followcount",
}
col, ok := allowedSortColumns[req.Sort]
if !ok {
    col = "c.updatedat"   // default
}
// col is safe — it came from our whitelist, not user input
query := fmt.Sprintf(`SELECT ... FROM core.comic c ORDER BY %s DESC`, col)
```

---

## 6. Secrets Management

### Required environment variables

| Variable | Type | Rotation |
|---|---|---|
| `JWT_PRIVATE_KEY` / `JWT_PRIVATE_KEY_PATH` | RSA-2048 PEM | Annually or on compromise |
| `JWT_PUBLIC_KEY` / `JWT_PUBLIC_KEY_PATH` | RSA-2048 PEM (public) | With private key |
| `UNSUBSCRIBE_SECRET` | 32-byte random hex | Annually |
| `DATABASE_URL` | PostgreSQL DSN | On compromise |
| `REDIS_URL` | Redis DSN | On compromise |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | Object storage credentials | Quarterly |
| `RESEND_API_KEY` / `SENDGRID_API_KEY` | SMTP provider API key | On compromise |

### Never commit secrets

```gitignore
# .gitignore — must include
.env
.env.*
*.pem
jwt_private.pem
config/secrets.yaml
```

### OAuth token encryption at rest

```go
// users.oauthprovider.accesstoken must be encrypted at rest
// Use AES-256-GCM with key from env

func encrypt(plaintext, key []byte) (string, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

---

## 7. Data Privacy

### IP address anonymization

```go
// analytics.pageview and users.session store IP addresses for:
// - Rate limiting
// - Fraud detection
// - Support investigations
//
// After 90 days: IP and User-Agent are SET to NULL by background job
// (see BATCH_API.md → analytics.anonymize)
```

### PII fields

| Field | Table | Purpose | Retention |
|---|---|---|---|
| `email` | `users.account` | Login, notifications | Until account deleted |
| `ipaddress` | `analytics.pageview` | Analytics | 90 days (anonymized) |
| `ipaddress` | `users.session` | Security/fraud | Session lifetime |
| `useragent` | `analytics.pageview` | Analytics | 90 days (anonymized) |
| `useragent` | `users.session` | Device display | Session lifetime |

### Account deletion

```go
// Soft delete: deletedat = NOW()
// PII fields zeroed out immediately on delete request:
UPDATE users.account
SET
    email       = 'deleted-' || id || '@yomira.invalid',
    passwordhash = NULL,
    displayname  = NULL,
    avatarurl    = NULL,
    bio          = NULL,
    website      = NULL,
    deletedat    = NOW()
WHERE id = $1;
-- Note: username is kept (prevents username squatting / confusion)
-- All sessions revoked
-- OAuth providers cascade-deleted
```

---

## 8. HTTP Security Headers

Applied by the Go middleware on every response:

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        h := w.Header()
        // Prevent MIME type sniffing
        h.Set("X-Content-Type-Options", "nosniff")
        // Prevent clickjacking (API — not serving HTML)
        h.Set("X-Frame-Options", "DENY")
        // HSTS — force HTTPS for 1 year
        h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        // Minimal referrer information
        h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        // Content Security Policy (API only — no HTML served)
        h.Set("Content-Security-Policy", "default-src 'none'")
        // Remove server fingerprint
        h.Set("Server", "")

        next.ServeHTTP(w, r)
    })
}
```

---

## 9. Rate Limiting & Abuse Prevention

See [MIDDLEWARE.md §7](./MIDDLEWARE.md#7-rate-limiter) for the rate limiter implementation.

### Endpoint-specific limits

| Endpoint | Limit | Window | Key |
|---|---|---|---|
| `POST /auth/login` | 10 req | 15 min | Per IP |
| `POST /auth/register` | 5 req | 15 min | Per IP |
| `POST /auth/forgot-password` | 3 req | 15 min | Per email |
| `POST /auth/email/verify/send` | 3 req | 15 min | Per email |
| `POST /me/email/change/request` | 3 req | 24 hours | Per user |
| `POST /upload/*` | 10 req | 1 min | Per user |
| `GET /search/quick` | 30 req | 1 min | Per IP |
| Global authenticated | 300 req | 1 min | Per user |
| Global unauthenticated | 60 req | 1 min | Per IP |

---

## 10. Dependency Security

```bash
# Audit Go modules for known CVEs
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Run weekly in CI:
# .github/workflows/security.yml
- name: Security audit
  run: govulncheck ./...
```

**Dependency rules:**
- Keep dependencies minimal — each adds attack surface
- Pin exact versions in `go.sum`
- Review `go mod tidy` diffs in PRs
- No dependencies with `replace` directives pointing to untrusted forks

---

## 11. Security Checklist

### Pre-deployment

- [ ] All secrets in env vars / secret manager — not in code
- [ ] `JWT_PRIVATE_KEY` is RSA-2048+ (not HS256 shared secret)
- [ ] `bcrypt` cost = 12 in production config
- [ ] OAuth `accesstoken` encrypted at rest
- [ ] Database connection uses TLS (`sslmode=require`)
- [ ] Redis connection uses TLS (if not on private network)
- [ ] Object storage bucket not publicly writable
- [ ] `CORS` allowedOrigins does not include `*`
- [ ] Rate limiter connected to Redis (not in-memory for multi-instance)
- [ ] `robots.txt` served and up to date
- [ ] `govulncheck` passes in CI

### Code review checklist

- [ ] No raw SQL string concatenation — only `$1`, `$2` parameters
- [ ] Owner check on all `PATCH` / `DELETE` mutations
- [ ] No secrets in log output or error messages
- [ ] Request struct fields explicitly whitelisted (no mass assignment)
- [ ] File uploads validated for format, size, and MIME type — not just extension
- [ ] Redirect URLs validated against allowlist (OAuth callback)
- [ ] Tokens/nonces are cryptographically random (`crypto/rand`, not `math/rand`)
