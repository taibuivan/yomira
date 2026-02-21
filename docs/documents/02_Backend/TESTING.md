# Testing Guide — Yomira Backend

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22  
> **Applies to:** `src/server/`, `tests/`

---

## Table of Contents

1. [Testing Philosophy](#1-testing-philosophy)
2. [Test Types & When to Use](#2-test-types--when-to-use)
3. [Unit Tests](#3-unit-tests)
4. [Integration Tests](#4-integration-tests)
5. [HTTP Handler Tests](#5-http-handler-tests)
6. [Middleware Tests](#6-middleware-tests)
7. [Test Utilities & Helpers](#7-test-utilities--helpers)
8. [Mocking Pattern](#8-mocking-pattern)
9. [Running Tests](#9-running-tests)
10. [Coverage Targets](#10-coverage-targets)
11. [CI Integration](#11-ci-integration)

---

## 1. Testing Philosophy

1. **Test behavior, not implementation** — test what a function does, not how.
2. **Integration tests over unit tests for storage** — the SQL itself matters; mock the DB sparingly.
3. **Table-driven tests** — Go convention: one `tests := []struct{...}` slice per test function.
4. **No `t.Skip()`** in CI — all tests must run. Mark flaky tests as known issues in GitHub.
5. **Fast tests** — unit tests < 1ms each; integration tests < 5s total with testcontainers.
6. **Test the error path too** — every sentinel error should have a test case.

---

## 2. Test Types & When to Use

| Type | Speed | DB | Use for |
|---|---|---|---|
| **Unit** | < 1ms | Mock (interface) | Service business logic, validators, utility functions |
| **Integration** | ~3s startup | Real PostgreSQL via testcontainers | Storage (SQL correctness), migrations |
| **Handler (HTTP)** | < 5ms | Mock service | HTTP response codes, JSON shape, auth enforcement |
| **Middleware** | < 1ms | None | Middleware behavior (auth, CORS, rate limit) |
| **E2E** | Slow | Staging DB | Critical user flows (manual or CI nightly) |

---

## 3. Unit Tests

### File location

```
src/server/service/comic/service.go
src/server/service/comic/service_test.go   ← same package (_test suffix optional)
```

### Table-driven test (service layer)

```go
// src/server/service/comic/service_test.go
func TestComicService_GetByID(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        mockFn  func(m *mockComicRepo)
        want    *Comic
        wantErr error
    }{
        {
            name: "returns comic",
            id:   "01952fa3-0000-7000-0000-000000000001",
            mockFn: func(m *mockComicRepo) {
                m.On("FindByID", mock.Anything, "01952fa3-0000-7000-0000-000000000001").
                    Return(&Comic{ID: "01952fa3-...", Title: "Solo Leveling"}, nil)
            },
            want: &Comic{Title: "Solo Leveling"},
        },
        {
            name: "not found",
            id:   "00000000-0000-7000-0000-000000000000",
            mockFn: func(m *mockComicRepo) {
                m.On("FindByID", mock.Anything, mock.Anything).
                    Return(nil, storage_comic.ErrComicNotFound)
            },
            wantErr: storage_comic.ErrComicNotFound,
        },
        {
            name: "deleted comic returns not found",
            id:   "01952fa3-0000-7000-0000-000000000002",
            mockFn: func(m *mockComicRepo) {
                m.On("FindByID", mock.Anything, mock.Anything).
                    Return(nil, storage_comic.ErrComicNotFound)  // repo hides deleted
            },
            wantErr: storage_comic.ErrComicNotFound,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            repo := &mockComicRepo{}
            tc.mockFn(repo)

            svc := NewComicService(repo)
            got, err := svc.GetByID(context.Background(), tc.id)

            assert.ErrorIs(t, err, tc.wantErr)
            if tc.want != nil {
                assert.Equal(t, tc.want.Title, got.Title)
            }
            repo.AssertExpectations(t)
        })
    }
}
```

### Validator unit test

```go
// src/server/shared/validate/validate_test.go
func TestValidator_Email(t *testing.T) {
    tests := []struct {
        input   string
        wantErr bool
    }{
        {"tai@example.com", false},
        {"tai.buivan.jp@gmail.com", false},
        {"notanemail", true},
        {"@nodomain.com", true},
        {"", true},
    }
    for _, tc := range tests {
        t.Run(tc.input, func(t *testing.T) {
            v := &Validator{}
            v.Email("email", tc.input)
            err := v.Err()
            if tc.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## 4. Integration Tests

Uses **testcontainers-go** to spin up real PostgreSQL. Tests run migrations and exercises actual SQL.

### Setup (shared across all integration tests)

```go
// tests/integration/testdb/testdb.go
package testdb

import (
    "context"
    "testing"

    "github.com/golang-migrate/migrate/v4"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
    once sync.Once
    dsn  string
)

// Setup starts a PostgreSQL container and runs migrations.
// Call from TestMain. Container is shared across all tests in the package.
func Setup(t *testing.T) (*sql.DB, string) {
    t.Helper()
    var db *sql.DB

    once.Do(func() {
        ctx := context.Background()
        pg, err := postgres.RunContainer(ctx,
            testcontainers.WithImage("postgres:16"),
            postgres.WithDatabase("yomira_test"),
            postgres.WithUsername("yomira"),
            postgres.WithPassword("test"),
        )
        require.NoError(t, err)

        dsn, _ = pg.ConnectionString(ctx, "sslmode=disable")

        // Run all migrations
        m, _ := migrate.New(
            "file://../../src/common/DML/migrations",
            dsn,
        )
        require.NoError(t, m.Up())

        db, _ = sql.Open("pgx", dsn)
        t.Cleanup(func() { pg.Terminate(ctx) })
    })

    return db, dsn
}

// CleanTable truncates a table and resets sequences between tests.
func CleanTable(t *testing.T, db *sql.DB, tables ...string) {
    t.Helper()
    for _, table := range tables {
        db.Exec("TRUNCATE " + table + " RESTART IDENTITY CASCADE")
    }
}
```

### Storage integration test

```go
// tests/integration/comic_repo_test.go
func TestMain(m *testing.M) {
    db, _ = testdb.Setup(&testing.T{})
    os.Exit(m.Run())
}

func TestComicRepo_FindByID(t *testing.T) {
    testdb.CleanTable(t, db, "core.comic")
    repo := comic.NewRepo(db)

    // Insert test data
    id := uuidv7.New().String()
    db.Exec(`INSERT INTO core.comic (id, title, slug, status)
             VALUES ($1, 'Test Comic', 'test-comic', 'ongoing')`, id)

    t.Run("found", func(t *testing.T) {
        got, err := repo.FindByID(context.Background(), id)
        require.NoError(t, err)
        assert.Equal(t, "Test Comic", got.Title)
    })

    t.Run("not found", func(t *testing.T) {
        _, err := repo.FindByID(context.Background(), "00000000-0000-7000-0000-000000000000")
        assert.ErrorIs(t, err, comic.ErrComicNotFound)
    })

    t.Run("soft deleted not found", func(t *testing.T) {
        db.Exec(`UPDATE core.comic SET deletedat = NOW() WHERE id = $1`, id)
        _, err := repo.FindByID(context.Background(), id)
        assert.ErrorIs(t, err, comic.ErrComicNotFound)
    })
}

func TestComicRepo_Upsert_Rating(t *testing.T) {
    // Tests ON CONFLICT DO UPDATE SQL
    repo := comic.NewRepo(db)

    err := repo.UpsertRating(context.Background(), userID, comicID, 9)
    require.NoError(t, err)

    // Update (same user, same comic)
    err = repo.UpsertRating(context.Background(), userID, comicID, 10)
    require.NoError(t, err)

    score, _ := repo.GetUserRating(context.Background(), userID, comicID)
    assert.Equal(t, 10, score, "should be updated to 10")
}
```

---

## 5. HTTP Handler Tests

Use `httptest` to test full handler → service → (mock storage) stack.

```go
// src/server/api/comic/handler_test.go
func TestComicHandler_GetByID(t *testing.T) {
    tests := []struct {
        name       string
        comicID    string
        setupMock  func(svc *mockComicService)
        wantStatus int
        wantCode   string
    }{
        {
            name:    "returns 200 with comic",
            comicID: validUUID,
            setupMock: func(svc *mockComicService) {
                svc.On("GetByID", mock.Anything, validUUID).
                    Return(&Comic{ID: validUUID, Title: "Test"}, nil)
            },
            wantStatus: http.StatusOK,
        },
        {
            name:    "returns 404 for unknown comic",
            comicID: unknownUUID,
            setupMock: func(svc *mockComicService) {
                svc.On("GetByID", mock.Anything, unknownUUID).
                    Return(nil, ErrComicNotFound)
            },
            wantStatus: http.StatusNotFound,
            wantCode:   "NOT_FOUND",
        },
        {
            name:    "returns 401 without auth on protected endpoint",
            comicID: validUUID,
            setupMock: func(svc *mockComicService) {},   // not called
            wantStatus: http.StatusUnauthorized,
            wantCode:   "UNAUTHORIZED",
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            svc := &mockComicService{}
            tc.setupMock(svc)

            handler := NewComicHandler(svc)
            r := chi.NewRouter()
            r.Get("/comics/{id}", handler.GetByID)

            req := httptest.NewRequest(http.MethodGet, "/comics/"+tc.comicID, nil)
            rr := httptest.NewRecorder()
            r.ServeHTTP(rr, req)

            assert.Equal(t, tc.wantStatus, rr.Code)
            if tc.wantCode != "" {
                var body map[string]string
                json.NewDecoder(rr.Body).Decode(&body)
                assert.Equal(t, tc.wantCode, body["code"])
            }
            svc.AssertExpectations(t)
        })
    }
}
```

---

## 6. Middleware Tests

```go
// src/server/middleware/auth_test.go
func TestAuthMiddleware(t *testing.T) {
    verifier := newTestVerifier(t)   // creates real RS256 verifier with test keys

    tests := []struct {
        name       string
        authHeader string
        wantStatus int
        wantCode   string
    }{
        {"valid token", "Bearer " + validToken(t), http.StatusOK, ""},
        {"missing header", "", http.StatusUnauthorized, "UNAUTHORIZED"},
        {"malformed token", "Bearer not.a.jwt", http.StatusUnauthorized, "TOKEN_INVALID"},
        {"expired token", "Bearer " + expiredToken(t), http.StatusUnauthorized, "TOKEN_EXPIRED"},
        {"wrong prefix", "Token " + validToken(t), http.StatusUnauthorized, "UNAUTHORIZED"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            reached := false
            handler := Auth(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                reached = true
                w.WriteHeader(http.StatusOK)
            }))

            req := httptest.NewRequest(http.MethodGet, "/", nil)
            if tc.authHeader != "" {
                req.Header.Set("Authorization", tc.authHeader)
            }
            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)

            assert.Equal(t, tc.wantStatus, rr.Code)
            if tc.wantCode != "" {
                var body map[string]string
                json.Unmarshal(rr.Body.Bytes(), &body)
                assert.Equal(t, tc.wantCode, body["code"])
            }
            if tc.wantStatus == http.StatusOK {
                assert.True(t, reached, "handler should have been called")
            }
        })
    }
}
```

---

## 7. Test Utilities & Helpers

```go
// tests/testutil/auth.go
// Create a valid signed JWT for testing
func ValidToken(t *testing.T, userID, role string, verified bool) string {
    t.Helper()
    claims := auth.Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID,
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
        },
        Role: role, IsVerified: verified,
    }
    tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).
        SignedString(testPrivateKey)
    require.NoError(t, err)
    return tok
}

// tests/testutil/request.go
// Build an authenticated request
func AuthRequest(t *testing.T, method, path string, body any, userID, role string) *http.Request {
    t.Helper()
    var buf bytes.Buffer
    if body != nil {
        json.NewEncoder(&buf).Encode(body)
    }
    req := httptest.NewRequest(method, path, &buf)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+ValidToken(t, userID, role, true))
    return req
}

// tests/testutil/assert.go
// Assert JSON response shape
func AssertJSONCode(t *testing.T, rr *httptest.ResponseRecorder, wantCode string) {
    t.Helper()
    var body map[string]string
    require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
    assert.Equal(t, wantCode, body["code"])
}
```

---

## 8. Mocking Pattern

Use `testify/mock` for interface mocks. Generate with `mockery` (optional).

```go
// src/server/service/comic/mock_repo_test.go
type mockComicRepo struct{ mock.Mock }

func (m *mockComicRepo) FindByID(ctx context.Context, id string) (*Comic, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Comic), args.Error(1)
}

func (m *mockComicRepo) UpsertRating(ctx context.Context, userID, comicID string, score int) error {
    return m.Called(ctx, userID, comicID, score).Error(0)
}
```

### Mockery (auto-generate mocks)

```bash
go install github.com/vektra/mockery/v2@latest

# Generate mock for ComicRepository interface
mockery --name=ComicRepository --dir=src/server/storage/comic \
        --output=src/server/storage/comic/mocks --outpkg=mocks
```

---

## 9. Running Tests

```bash
# All tests
go test ./...

# With race detector (required in CI)
go test -race ./...

# Single package
go test ./src/server/service/comic/...

# Single test function
go test -run TestComicService_GetByID ./src/server/service/comic/...

# With verbose output
go test -v -run TestAuthMiddleware ./src/server/middleware/...

# Integration tests only (requires Docker)
go test -tags=integration ./tests/integration/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out   # open in browser
```

---

## 10. Coverage Targets

| Layer | Target | Notes |
|---|---|---|
| `shared/` (validate, apperr, respond) | ≥ 90% | Pure functions — easy to test exhaustively |
| `middleware/` | ≥ 85% | All error paths + happy path |
| `service/` | ≥ 75% | Business logic — most important layer |
| `storage/` (integration) | ≥ 65% | Covers SQL correctness |
| `api/` (handlers) | ≥ 60% | HTTP contract — status codes, JSON shape |

```bash
# Check coverage by package
go test -coverprofile=coverage.out ./... && \
go tool cover -func=coverage.out | sort -k3 -n
```

---

## 11. CI Integration

```yaml
# .github/workflows/ci.yml (test step)
- name: Run tests with race detector
  run: go test -race -coverprofile=coverage.out ./...
  env:
    DATABASE_URL: postgres://yomira:test@localhost:5432/yomira_test?sslmode=disable
    REDIS_URL: redis://localhost:6379/0

- name: Check coverage thresholds
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Total coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 60" | bc -l) )); then
      echo "Coverage below 60% threshold"
      exit 1
    fi

- name: Upload to Codecov
  uses: codecov/codecov-action@v4
  with:
    file: coverage.out
```
