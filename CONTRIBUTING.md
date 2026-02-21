# Contributing to Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Local Setup](#2-local-setup)
3. [Project Structure](#3-project-structure)
4. [Code Style](#4-code-style)
5. [Commit Message Convention](#5-commit-message-convention)
6. [Branch Naming](#6-branch-naming)
7. [Pull Request Process](#7-pull-request-process)
8. [Writing Tests](#8-writing-tests)
9. [Adding a Migration](#9-adding-a-migration)
10. [CI Checks](#10-ci-checks)

---

## 1. Prerequisites

- Go ≥ 1.22
- Docker Desktop (for infrastructure)
- `golang-migrate` CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- `golangci-lint`: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- `air` (hot reload): `go install github.com/air-verse/air@latest`

---

## 2. Local Setup

See [OPERATIONS.md](docs/documents/05_Operations/OPERATIONS.md) for full setup instructions.

```bash
git clone https://github.com/your-org/yomira.git
cd yomira
cp .env.example .env
docker compose up -d postgres redis minio mailhog
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up
air
```

---

## 3. Project Structure

```
yomira/
├── cmd/
│   └── server/              # main entrypoint
├── src/
│   ├── common/
│   │   └── DML/             # SQL schemas + migrations
│   └── server/
│       ├── api/             # HTTP handlers
│       ├── service/         # business logic
│       ├── storage/         # database queries
│       ├── middleware/      # Go middleware chain
│       └── shared/          # apperr, validate, respond, auth helpers
├── docs/
│   └── documents/           # project documentation
├── .env.example
├── docker-compose.yml
└── .air.toml
```

**Layer rules:**
- `api` → depends on `service` only (never `storage` directly)
- `service` → depends on `storage` interfaces only (never concrete types)
- `storage` → depends on database driver only
- `shared` → no dependencies on other internal packages

---

## 4. Code Style

### Go formatting

```bash
# Format before committing (enforced by CI)
gofmt -w .
go vet ./...
golangci-lint run ./...
```

### Key lint rules (`.golangci.yml`)

- `errcheck` — no ignored errors (`_ = something` is not allowed)
- `govet` — shadow, nilness checks
- `staticcheck` — deprecated APIs, unnecessary conversions
- `revive` — exported functions must have comments
- `gocyclo` — max cyclomatic complexity 15

### Naming

```go
// ✅ Match Postgres column names exactly (all lowercase)
type Comic struct {
    ID              string    `json:"id" db:"id"`
    Title           string    `json:"title" db:"title"`
    FollowCount     int       `json:"followcount" db:"followcount"`
    RatingBayesian  float64   `json:"ratingbayesian" db:"ratingbayesian"`
}

// ✅ Interface names end in -er or -Repository
type ComicRepository interface { ... }
type ComicService interface { ... }

// ✅ Error variables start with Err
var ErrComicNotFound = apperr.NotFound("Comic")

// ✅ Context is always first parameter
func (s *ComicService) GetByID(ctx context.Context, id string) (*Comic, error)
```

### Comments

```go
// ✅ Exported functions must have doc comment
// GetByID returns a single comic by its UUIDv7 ID.
// Returns ErrComicNotFound if the comic does not exist or is soft-deleted.
func (s *ComicService) GetByID(ctx context.Context, id string) (*Comic, error)

// ✅ Complex SQL should have a comment explaining why
// Why: COUNT(*) OVER() avoids a second round-trip for total count.
rows, err := r.db.QueryContext(ctx, `
    SELECT id, title, COUNT(*) OVER() AS total FROM core.comic ...
`)
```

---

## 5. Commit Message Convention

Follow **Conventional Commits** (enforced by CI commit-msg hook):

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

### Types

| Type | Use for |
|---|---|
| `feat` | New feature or endpoint |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change without feature or fix |
| `test` | Adding or updating tests |
| `chore` | Build, deps, CI, tooling |
| `perf` | Performance improvement |
| `sql` | Schema change (migration) |

### Scope (optional)

`users`, `core`, `library`, `social`, `crawler`, `analytics`, `system`, `middleware`, `auth`, `ci`

### Examples

```
feat(library): add bulk mark-all-read endpoint

fix(auth): prevent refresh token reuse after revocation

docs(api): add SEARCH_API.md

sql(analytics): add trigram index on comic title

chore(deps): update pgx to v5.5.4

test(social): add comment vote integration tests
```

### Rules

- Subject line: max 72 chars, lowercase, no period at end
- Use imperative mood: "add" not "added" / "adds"
- Body: explain *why*, not *what*. The diff shows the what.

---

## 6. Branch Naming

```
<type>/<short-description>

feat/bulk-mark-read
fix/refresh-token-reuse
docs/search-api
sql/add-trigram-index
chore/update-pgx
```

- All lowercase, hyphens only
- Max 50 chars
- Reference issue number if applicable: `feat/123-bulk-mark-read`

---

## 7. Pull Request Process

### Before opening a PR

```bash
# 1. Sync with main
git fetch origin && git rebase origin/main

# 2. Run all checks locally
gofmt -w .
go vet ./...
golangci-lint run ./...
go test ./...

# 3. If you added a migration, test up AND down
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
```

### PR checklist

- [ ] Title follows commit convention (`feat(scope): description`)
- [ ] Description explains *why* this change is needed
- [ ] Tests added or updated for new behavior
- [ ] Migration has both `up` and `down` files
- [ ] No secrets or `.env` files committed
- [ ] `CHANGELOG.md` updated if schema changed
- [ ] API docs updated if endpoints changed
- [ ] CI checks pass

### Review expectations

- At least **1 approval** required before merge
- Address all review comments or explain why not
- Squash merge preferred (keep history clean)
- No force-push to `main`

---

## 8. Writing Tests

### Unit tests

```go
// storage/comic/repo_test.go
// Use table-driven tests
func TestComicRepo_FindByID(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        wantErr error
    }{
        {"found", validComicID, nil},
        {"not found", "00000000-0000-0000-0000-000000000000", ErrComicNotFound},
        {"deleted", deletedComicID, ErrComicNotFound},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            comic, err := repo.FindByID(ctx, tc.id)
            assert.ErrorIs(t, err, tc.wantErr)
            if tc.wantErr == nil {
                assert.NotNil(t, comic)
            }
        })
    }
}
```

### Integration tests (testcontainers)

```go
// tests/integration/setup_test.go
func TestMain(m *testing.M) {
    ctx := context.Background()
    pg, _ := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16"),
        postgres.WithDatabase("yomira_test"),
    )
    dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")

    // Run migrations
    mig, _ := migrate.New("file://../../src/common/DML/migrations", dsn)
    mig.Up()

    os.Exit(m.Run())
}
```

### Test file conventions

```
src/storage/comic/repo.go
src/storage/comic/repo_test.go      ← unit test (same package, mock DB)
tests/integration/comic_test.go     ← integration test (real DB via testcontainers)
```

### Coverage target

- Storage layer: ≥ 70%
- Service layer: ≥ 80%
- Middleware: ≥ 90%

---

## 9. Adding a Migration

See [MIGRATION_GUIDE.md](docs/documents/03_Database/MIGRATION_GUIDE.md) for full instructions.

```bash
# 1. Create migration files
touch src/common/DML/migrations/000XXX_your_description.up.sql
touch src/common/DML/migrations/000XXX_your_description.down.sql

# 2. Write SQL (up = apply, down = reverse)

# 3. Test locally
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
# verify change
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations down 1
# verify rollback
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up 1
# re-apply for development

# 4. Update CHANGELOG.md in DML directory
# 5. Update the source SQL file (10_USERS/USERS.sql, etc.) to reflect new state
```

---

## 10. CI Checks

The following checks run on every PR (`.github/workflows/ci.yml`):

| Check | Command | Must pass |
|---|---|---|
| Format | `gofmt -l .` | Yes |
| Lint | `golangci-lint run ./...` | Yes |
| Vet | `go vet ./...` | Yes |
| Unit tests | `go test -race ./...` | Yes |
| Security audit | `govulncheck ./...` | Yes |
| Migration up/down | `migrate up && migrate down 1 && migrate up` | Yes |
| Build | `go build ./cmd/server` | Yes |

### Running CI checks locally

```bash
# All checks in one go (mirrors CI)
gofmt -l . && \
go vet ./... && \
golangci-lint run ./... && \
go test -race ./... && \
govulncheck ./... && \
go build ./cmd/server && \
echo "✅ All checks passed"
```
