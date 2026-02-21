# Go Error Handling — Yomira Backend

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Applies to:** `src/server/`, `src/service/`, `src/storage/`

---

## Table of Contents

1. [Design Goals](#1-design-goals)
2. [Error Type System](#2-error-type-system)
3. [Sentinel Errors](#3-sentinel-errors)
4. [Error Wrapping](#4-error-wrapping)
5. [Service → Handler Mapping](#5-service--handler-mapping)
6. [Validation Errors](#6-validation-errors)
7. [Database Error Handling](#7-database-error-handling)
8. [Logging Strategy](#8-logging-strategy)
9. [Do's and Don'ts](#9-dos-and-donts)

---

## 1. Design Goals

1. **Never panic** on expected errors — panics are only for programming bugs.
2. **Never expose internal details** to the client — wrap errors carefully.
3. **Always log at the error origin** — not at every layer above it.
4. **One error type per condition** — sentinel errors are the canonical way to signal a specific failure.
5. **Consistent HTTP response** — all error responses follow the `{ "error": "...", "code": "..." }` envelope.

---

## 2. Error Type System

```go
// shared/apperr/apperr.go

// AppError is the central error type.
// Every domain error eventually becomes an AppError before reaching the HTTP layer.
type AppError struct {
    Code       string        // machine-readable, matches API convention (e.g. "NOT_FOUND")
    Message    string        // human-readable, safe to send to client
    HTTPStatus int           // HTTP status code
    Cause      error         // original error (for logging only, never sent to client)
    Details    []FieldError  // per-field validation errors (optional)
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Cause }

// FieldError represents a single field-level validation failure
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

### Constructors

```go
func NotFound(resource string) *AppError {
    return &AppError{
        Code:       "NOT_FOUND",
        Message:    resource + " not found",
        HTTPStatus: http.StatusNotFound,
    }
}

func Forbidden(msg string) *AppError {
    return &AppError{
        Code:       "FORBIDDEN",
        Message:    msg,
        HTTPStatus: http.StatusForbidden,
    }
}

func Conflict(msg string) *AppError {
    return &AppError{
        Code:       "CONFLICT",
        Message:    msg,
        HTTPStatus: http.StatusConflict,
    }
}

func ValidationError(msg string, details ...FieldError) *AppError {
    return &AppError{
        Code:       "VALIDATION_ERROR",
        Message:    msg,
        HTTPStatus: http.StatusBadRequest,
        Details:    details,
    }
}

func Internal(cause error) *AppError {
    return &AppError{
        Code:       "INTERNAL_ERROR",
        Message:    "An unexpected error occurred",
        HTTPStatus: http.StatusInternalServerError,
        Cause:      cause,
    }
}

func RateLimited(retryAfterSeconds int) *AppError {
    return &AppError{
        Code:       "RATE_LIMITED",
        Message:    fmt.Sprintf("Too many requests. Try again in %ds.", retryAfterSeconds),
        HTTPStatus: http.StatusTooManyRequests,
    }
}
```

---

## 3. Sentinel Errors

Sentinel errors are **package-level vars** declared in each domain package. Handlers `errors.Is()` against them.

```go
// storage/comic/errors.go
var (
    ErrComicNotFound    = apperr.NotFound("Comic")
    ErrComicLocked      = apperr.Forbidden("Comic is locked for moderation")
    ErrComicDeleted     = apperr.NotFound("Comic")         // same HTTP, different sentinel
    ErrDuplicateSlug    = apperr.Conflict("A comic with this slug already exists")
)

// storage/user/errors.go
var (
    ErrUserNotFound       = apperr.NotFound("User")
    ErrDuplicateEmail     = &apperr.AppError{Code: "DUPLICATE_EMAIL",   HTTPStatus: 409, Message: "Email already registered"}
    ErrDuplicateUsername  = &apperr.AppError{Code: "DUPLICATE_USERNAME", HTTPStatus: 409, Message: "Username already taken"}
    ErrAccountSuspended   = &apperr.AppError{Code: "ACCOUNT_SUSPENDED",  HTTPStatus: 403, Message: "Account is suspended"}
    ErrEmailNotVerified   = &apperr.AppError{Code: "EMAIL_NOT_VERIFIED", HTTPStatus: 403, Message: "Email not verified"}
    ErrInvalidCredentials = &apperr.AppError{Code: "INVALID_CREDENTIALS",HTTPStatus: 400, Message: "Invalid email or password"}
)

// storage/library/errors.go
var (
    ErrEntryNotFound     = apperr.NotFound("Library entry")
    ErrEntryAlreadyExists = apperr.Conflict("Comic already in your library")
    ErrListNotFound      = apperr.NotFound("Custom list")
    ErrListFull          = &apperr.AppError{Code: "LIMIT_EXCEEDED", HTTPStatus: 422,
                              Message: "Custom list is full (500 items maximum)"}
)
```

---

## 4. Error Wrapping

Use `fmt.Errorf("context: %w", err)` to add context while preserving the sentinel for `errors.Is()`.

```go
// storage/comic/repo.go
func (r *ComicRepo) FindByID(ctx context.Context, id string) (*Comic, error) {
    var comic Comic
    err := r.db.QueryRowContext(ctx, `SELECT ... FROM core.comic WHERE id = $1`, id).
        Scan(&comic.ID, &comic.Title ...)

    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrComicNotFound      // return sentinel directly — no wrapping needed
    }
    if err != nil {
        return nil, fmt.Errorf("ComicRepo.FindByID: %w", err)   // wrap unexpected errors
    }
    return &comic, nil
}
```

```go
// service/comic/service.go
func (s *ComicService) GetByID(ctx context.Context, id string) (*Comic, error) {
    comic, err := s.repo.FindByID(ctx, id)
    if err != nil {
        // Don't re-wrap AppErrors — they're already correctly typed
        if apperr.IsAppError(err) {
            return nil, err
        }
        // Wrap unexpected storage errors as internal
        return nil, apperr.Internal(fmt.Errorf("ComicService.GetByID: %w", err))
    }
    return comic, nil
}
```

---

## 5. Service → Handler Mapping

The HTTP handler is the **only place** that converts errors to HTTP responses. Services and storage layers return Go errors — never write HTTP responses there.

```go
// handler/comic/handler.go
func (h *ComicHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    comic, err := h.svc.GetByID(r.Context(), id)
    if err != nil {
        RespondError(w, r, err)   // central error → HTTP converter
        return
    }
    RespondJSON(w, http.StatusOK, map[string]any{"data": comic})
}

// shared/respond/respond.go
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
    var appErr *apperr.AppError
    if !errors.As(err, &appErr) {
        // Unexpected error — log and mask
        log := GetLogger(r.Context())
        log.ErrorContext(r.Context(), "unhandled error",
            slog.String("error", err.Error()),
            slog.String("request_id", GetRequestID(r.Context())))
        appErr = apperr.Internal(err)
    }

    // Log 5xx errors with cause; 4xx are expected — log at WARN
    if appErr.HTTPStatus >= 500 {
        log := GetLogger(r.Context())
        log.ErrorContext(r.Context(), "server error",
            slog.String("code",       appErr.Code),
            slog.String("request_id", GetRequestID(r.Context())),
            slog.Any("cause",         appErr.Cause))
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(appErr.HTTPStatus)

    body := map[string]any{
        "error": appErr.Message,
        "code":  appErr.Code,
    }
    if len(appErr.Details) > 0 {
        body["details"] = appErr.Details
    }
    _ = json.NewEncoder(w).Encode(body)
}
```

---

## 6. Validation Errors

Use a dedicated `Validator` struct to collect multiple field errors before returning.

```go
// shared/validate/validate.go
type Validator struct {
    errs []apperr.FieldError
}

func (v *Validator) Required(field, value string) {
    if strings.TrimSpace(value) == "" {
        v.errs = append(v.errs, apperr.FieldError{
            Field: field, Message: "This field is required",
        })
    }
}

func (v *Validator) MaxLen(field, value string, max int) {
    if len([]rune(value)) > max {
        v.errs = append(v.errs, apperr.FieldError{
            Field: field, Message: fmt.Sprintf("Maximum %d characters", max),
        })
    }
}

func (v *Validator) Range(field string, value, min, max int) {
    if value < min || value > max {
        v.errs = append(v.errs, apperr.FieldError{
            Field: field, Message: fmt.Sprintf("Must be between %d and %d", min, max),
        })
    }
}

func (v *Validator) Email(field, value string) {
    if !emailRegex.MatchString(value) {
        v.errs = append(v.errs, apperr.FieldError{
            Field: field, Message: "Must be a valid email address",
        })
    }
}

func (v *Validator) Err() error {
    if len(v.errs) == 0 {
        return nil
    }
    return apperr.ValidationError("Validation failed", v.errs...)
}
```

### Usage in service

```go
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*User, error) {
    v := &validate.Validator{}
    v.Required("username", req.Username)
    v.MaxLen("username", req.Username, 64)
    v.Required("email", req.Email)
    v.Email("email", req.Email)
    v.Required("password", req.Password)
    v.MinLen("password", req.Password, 8)
    if err := v.Err(); err != nil {
        return nil, err     // returns AppError with Details array
    }
    // ... proceed
}
```

---

## 7. Database Error Handling

PostgreSQL errors are translated at the **storage layer** — never let `pq` errors bubble up to service or handler.

```go
// shared/dberr/dberr.go
import "github.com/jackc/pgx/v5/pgconn"

func Translate(err error, sentinels ...error) error {
    if err == nil {
        return nil
    }

    // Check no-rows first (most common)
    if errors.Is(err, pgx.ErrNoRows) {
        if len(sentinels) > 0 {
            return sentinels[0]     // caller-provided "not found" sentinel
        }
        return apperr.NotFound("Resource")
    }

    // Postgres error codes
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505":  // unique_violation
            return translateUniqueViolation(pgErr)
        case "23503":  // foreign_key_violation
            return apperr.NotFound("Related resource")
        case "23514":  // check_violation
            return apperr.ValidationError("Value violates database constraint")
        case "40001":  // serialization_failure (retry)
            return apperr.Internal(fmt.Errorf("serialization_failure: %w", err))
        }
    }

    // Unknown error — wrap as internal
    return apperr.Internal(err)
}

func translateUniqueViolation(pgErr *pgconn.PgError) error {
    // Use constraint name to produce specific error
    switch pgErr.ConstraintName {
    case "account_email_uq":
        return storage_user.ErrDuplicateEmail
    case "account_username_uq":
        return storage_user.ErrDuplicateUsername
    case "comicsource_uq":
        return apperr.Conflict("This comic is already linked to this source with the same external ID")
    default:
        return apperr.Conflict("A duplicate entry already exists")
    }
}
```

---

## 8. Logging Strategy

| Error type | Log level | Log location | Context logged |
|---|---|---|---|
| `AppError` (4xx) | `WARN` | `RespondError` | `code`, `request_id` |
| `AppError` (5xx) | `ERROR` | `RespondError` | `code`, `cause`, `request_id` |
| DB error (translated) | `ERROR` | storage layer | full `pgErr`, query context |
| Panic | `ERROR` | `PanicRecovery` middleware | `stack_trace`, `request_id` |
| Auth failure | `WARN` | `Auth` middleware | `reason`, `ip`, `request_id` |
| Rate limit hit | `INFO` | `RateLimiter` middleware | `key`, `limit`, `ip` |

**Rule:** Log **once** at the origin. Do not re-log the same error as it propagates up. Use `err.Cause` in the log entry, not the wrapped message.

```go
// ✅ Correct — log at origin, return wrapped
func (r *ComicRepo) FindByID(ctx context.Context, id string) (*Comic, error) {
    err := r.db.QueryRow(ctx, q).Scan(...)
    if err != nil {
        slog.ErrorContext(ctx, "ComicRepo.FindByID DB error",
            slog.String("id", id),
            slog.Any("error", err))
        return nil, dberr.Translate(err, ErrComicNotFound)
    }
    return ...
}

// ❌ Wrong — don't log at every layer
func (s *ComicService) GetByID(ctx context.Context, id string) (*Comic, error) {
    comic, err := s.repo.FindByID(ctx, id)
    if err != nil {
        slog.ErrorContext(ctx, "GetByID failed", slog.Any("error", err))  // ← duplicate log
        return nil, err
    }
    return comic, nil
}
```

---

## 9. Do's and Don'ts

### ✅ Do

```go
// Return sentinel errors from storage
return nil, ErrComicNotFound

// Wrap unexpected errors with context
return nil, fmt.Errorf("ComicRepo.Create: %w", err)

// Use errors.Is() to check sentinels
if errors.Is(err, ErrComicNotFound) { ... }

// Use errors.As() to extract AppError
var appErr *apperr.AppError
if errors.As(err, &appErr) { ... }

// Validate all inputs at the service layer before hitting DB
v := &validate.Validator{}
v.Required("title", req.Title)
if err := v.Err(); err != nil { return nil, err }
```

### ❌ Don't

```go
// Never panic on expected errors
if comic == nil {
    panic("comic not found")   // ❌ use ErrComicNotFound instead
}

// Never expose internal error messages to the client
w.Write([]byte(err.Error()))   // ❌ may contain SQL query, file paths

// Never swallow errors silently
comic, _ := repo.FindByID(ctx, id)   // ❌ _ hides the error

// Never log the same error at multiple layers
slog.Error("failed", err)    // in repo
slog.Error("failed", err)    // in service — duplicate
slog.Error("failed", err)    // in handler — duplicate

// Never return raw pgx errors from the storage layer
return nil, pgxErr   // ❌ translate it first with dberr.Translate()
```
