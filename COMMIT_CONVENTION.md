# Commit Convention

Yomira follows the **[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)** specification, extended with project-specific scopes.

---

## Format

```
<type>(<scope>): <short description>

[optional body]

[optional footer(s)]
```

### Rules

| Part | Rule |
|---|---|
| **type** | Lowercase, from the allowed list below |
| **scope** | Lowercase, from the allowed list below; optional but strongly encouraged |
| **short description** | Imperative mood, lowercase, no trailing period, ≤ 72 characters |
| **body** | Plain text or bullet points; explain *why*, not *what*; wrap at 80 chars |
| **footer** | `BREAKING CHANGE:`, `Closes #<issue>`, `Refs #<issue>` |

---

## Allowed Types

| Type | When to Use |
|---|---|
| `feat` | A new feature visible to users or other services |
| `fix` | A bug fix |
| `perf` | Code change that improves performance, no functional change |
| `refactor` | Code restructuring with no feature or bug-fix change |
| `style` | Formatting, whitespace, naming — no logic change |
| `test` | Adding or updating tests only |
| `docs` | Documentation changes only |
| `chore` | Build process, dependency update, tooling, CI/CD |
| `ci` | Changes to CI configuration files or scripts |
| `revert` | Reverts a previous commit (include the reverted SHA in the body) |
| `wip` | Work-in-progress commit; **must not be merged to `main`** |

---

## Allowed Scopes

Scopes map to logical layers or packages within the monorepo.

| Scope | Area |
|---|---|
| `api` | Public API layer (`src/api/`) |
| `auth` | Authentication & authorization |
| `library` | Library & comic management |
| `reader` | Reading experience & chapter delivery |
| `sync` | Device sync & progress tracking |
| `crawler` | Web crawler / data collection |
| `extension` | Extension SDK (`src/extension-sdk/`) |
| `storage` | File storage, image handling |
| `db` | Database schema, migrations, queries |
| `config` | Application configuration |
| `web` | Web front-end (`src/apps/web/`) |
| `mobile` | Mobile app (`src/apps/mobile/`) |
| `common` | Shared utilities (`src/common/`) |
| `cmd` | Entry-point commands (`src/cmd/`) |
| `pkg` | Reusable packages (`src/pkg/`) |
| `docs` | Documentation only |
| `ci` | CI/CD pipelines |
| `deps` | Dependency bumps (`go.mod`, `package.json`, etc.) |
| `release` | Version tagging, changelog |

> You may omit the scope for purely cross-cutting changes.

---

## Breaking Changes

Append `!` after the type/scope **and** add a `BREAKING CHANGE:` footer:

```
feat(api)!: rename /comics endpoint to /library

BREAKING CHANGE: All clients must update their base URL from /v1/comics
to /v1/library. The old path returns 410 Gone.
```

---

## Examples

### Simple feature
```
feat(library): add reading-status filter to library listing
```

### Bug fix with issue reference
```
fix(reader): prevent duplicate page fetch on rapid swipe

Pages were being requested twice when the user swiped faster than the
debounce threshold. Added a request-in-flight guard to the page loader.

Closes #42
```

### Database migration
```
chore(db): add index on comic_chapters(comic_id, chapter_number)

Improves chapter list query time from ~120 ms to ~4 ms on a 50k-row
dataset. Migration file: 0007_add_chapter_index.sql.
```

### Refactor with no behavior change
```
refactor(common): extract pagination helper into pkg/pagination
```

### Documentation update
```
docs(api): document rate-limit headers in CORE_API.md
```

### Dependency bump
```
chore(deps): upgrade Go to 1.23 and update module dependencies
```

### Reverting a commit
```
revert: feat(crawler): add automatic retry on 429 responses

Reverts commit abc1234. The retry logic caused unbounded loops when
target sites returned persistent 429s with no Retry-After header.
```

---

## Branch Naming (companion convention)

```
<type>/<short-kebab-description>
```

Examples:
- `feat/library-status-filter`
- `fix/reader-duplicate-page-fetch`
- `chore/upgrade-go-1.23`
- `docs/api-rate-limit-headers`

---

## Git Hooks (recommended)

Install [`commitlint`](https://commitlint.js.org/) or [`pre-commit`](https://pre-commit.com/) to enforce this convention locally.  
A sample config will be added under `.githooks/` once tooling is set up.
