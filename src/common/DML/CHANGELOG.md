# DML Changelog

All schema changes are documented here in reverse-chronological order (newest first).

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).  
Version numbers follow `MAJOR.MINOR.PATCH`:

| Bump | When |
|---|---|
| **MAJOR** | Breaking change: table/column dropped, type changed, FK modified |
| **MINOR** | Additive: new table, new nullable column, new index |
| **PATCH** | Non-structural: comment update, default value tweak, seed data |

---

## [1.1.0] — 2026-02-21

### Added

| Table / Index | File | Description |
|---|---|---|
| `core.comictitle` | `20_CORE/CORE.sql` | Multilingual title + description per comic. Mirrors MangaDex language-map: `{"en":"...","ja":"..."}`. Fallback order in Go: user lang → `en` → `comic.title`. |
| `library.viewhistory` | `30_LIBRARY/LIBRARY.sql` | Browsing history (comic detail page views). Powers "Recently Viewed" in user profile. Capped at 500 rows/user by background job. Different scope from `readingprogress` (chapter-level). |
| `idx_core_chapter_group` | `20_CORE/CORE.sql` | `(scanlationgroupid, comicid, chapternumber) WHERE deletedat IS NULL` — optimises "all chapters by group X for comic Y" query used in group profile page. |
| `idx_library_entry_updatedat` | `30_LIBRARY/LIBRARY.sql` | `(userid, updatedat DESC)` — optimises "My Library" sort by recently updated. |
| `users.readingpreference.datasaver` | `10_USERS/USERS.sql` | `BOOLEAN NOT NULL DEFAULT FALSE` — enables low-bandwidth compressed image serving (MangaDex Data Saver feature). |

### Performance notes

- `idx_core_chapter_group` makes group-filtered chapter list ~10× faster on large comics (>1000 chapters from multiple groups).
- `idx_library_entry_updatedat` eliminates sequential scan on shelf sort-by-activity for users with >200 entries.

---


Version numbers follow `MAJOR.MINOR.PATCH`:

| Bump | When |
|---|---|
| **MAJOR** | Breaking change: table/column dropped, type changed, FK modified |
| **MINOR** | Additive: new table, new nullable column, new index |
| **PATCH** | Non-structural: comment update, default value tweak, seed data |

---

## [1.0.0] — 2026-02-21

> Initial release. Full PostgreSQL schema for **Yomira** — a manga/manhwa reading platform.

### Structure

Schema split into 9 focused files under numbered folders:

```
DML/
├── 00_SETUP/SETUP.sql        — extensions, schemas, shared trigger
├── 10_USERS/USERS.sql        — users schema (5 tables)
├── 20_CORE/CORE.sql          — core schema (18 tables)
├── 30_LIBRARY/LIBRARY.sql    — library schema (5 tables)
├── 40_SOCIAL/SOCIAL.sql      — social schema (12 tables + seed)
├── 50_CRAWLER/CRAWLER.sql    — crawler schema (4 tables)
├── 60_ANALYTICS/ANALYTICS.sql — analytics schema (2 tables, partitioned)
├── 70_SYSTEM/SYSTEM.sql      — system schema (3 tables)
└── DATA/INITIAL_DATA.sql     — seed / reference data
```

### Added — `users` schema (5 tables)

| Table | Description |
|---|---|
| `users.account` | Core user record. UUIDv7 PK. Soft-delete via `deletedat`. |
| `users.oauthprovider` | External OAuth identity links (Google, Discord, GitHub, Apple). |
| `users.session` | Active auth sessions. Stores SHA-256 token hashes only. |
| `users.follow` | Directed user-to-user follow graph. |
| `users.readingpreference` | Per-user reader UI settings (1:1 with account). |

### Added — `core` schema (18 tables)

| Table | Description |
|---|---|
| `core.language` | BCP-47 language reference. Static seed data. |
| `core.author` | Story writers. GIN trigram index on `name`. |
| `core.artist` | Illustrators. GIN trigram index on `name`. |
| `core.taggroup` | Tag category groupings: Genre / Theme / Format / Demographic. |
| `core.tag` | Unified tags replacing separate genre+tag tables. |
| `core.scanlationgroup` | Translation teams. Supports official publisher flag + social links. |
| `core.scanlationgroupmember` | Group membership with internal roles. |
| `core.scanlationgroupfollow` | Users following a scanlation group (feed source). |
| `core.comic` | Core content entity. Full-text `searchvector` via trigger. |
| `core.comicrelation` | Directed relation between comics (sequel, prequel, etc.). |
| `core.comicauthor` | M:N junction: comic ↔ author. |
| `core.comicartist` | M:N junction: comic ↔ artist. |
| `core.comictag` | M:N junction: comic ↔ tag (all groups). |
| `core.comiccover` | Per-volume cover images. |
| `core.comicart` | Art gallery: covers, fanart, promotional images. |
| `core.chapter` | Translated chapters. Multi-language per title. |
| `core.page` | Individual page images within a chapter. |
| `core.mediafile` | Object storage registry with SHA-256 deduplication. |

### Added — `library` schema (5 tables)

| Table | Description |
|---|---|
| `library.entry` | Personal comic shelf (reading status, score, last read). |
| `library.customlist` | User-curated named reading lists (public / private / unlisted). |
| `library.customlistitem` | Comics inside a custom list with manual sort order. |
| `library.readingprogress` | Last-read chapter + page per (user, comic). |
| `library.chapterread` | Append-only per-chapter completion record. |

### Added — `social` schema (12 tables)

| Table | Description |
|---|---|
| `social.comicrating` | Public 1–10 rating per (user, comic). |
| `social.comment` | Threaded comments on comics or chapters (XOR via Go). |
| `social.commentvote` | Up/down vote per (user, comment). |
| `social.notification` | Pull-based notification inbox. |
| `social.comicrecommendation` | User-submitted "also read" recommendations. |
| `social.comicrecommendationvote` | Votes on recommendations. |
| `social.feedevent` | Activity stream events (pull model). |
| `social.forum` | Forum boards (site-wide or per-comic). |
| `social.forumthread` | Discussion threads within a board. |
| `social.forumpost` | Individual replies within a thread. |
| `social.forumpostvote` | Up/down vote per (user, forumpost). |
| `social.report` | Content moderation report queue. |

### Added — `crawler` schema (4 tables)

| Table | Description |
|---|---|
| `crawler.source` | Registered external source websites (one per Go extension plugin). |
| `crawler.comicsource` | Maps a comic to its URL(s) on one or more sources. |
| `crawler.job` | Crawl job lifecycle record. |
| `crawler.log` | Append-only structured crawl logs. **Partitioned by month.** |

### Added — `analytics` schema (2 tables)

| Table | Description |
|---|---|
| `analytics.pageview` | Raw page view events. **Partitioned by month.** |
| `analytics.chaptersession` | Granular reading sessions (open/close timestamps). **Partitioned by month.** |

### Added — `system` schema (3 tables)

| Table | Description |
|---|---|
| `system.auditlog` | Immutable privileged-action audit log. Append-only. |
| `system.setting` | Global key-value configuration store. |
| `system.announcement` | Site-wide announcements. Soft-delete via `deletedat`. |

### Design Decisions (ADR)

- **Validation**: All enum-style allowed-value checks are enforced by the **Go service layer** exclusively. DB enforces only PK / UNIQUE / FK / NOT NULL.
- **IDs**: UUIDv7 (TEXT) for public/high-cardinality tables; SERIAL START 100000 for lookup tables.
- **Timestamps**: `createdat` + `updatedat` on all mutable tables. Trigger `set_updatedat()` auto-stamps `updatedat`. Append-only tables use `createdat` only.
- **Soft delete**: `deletedat TIMESTAMPTZ` — non-NULL = deleted. Partial indexes use `WHERE deletedat IS NULL`.
- **Full-text search**: `core.comic.searchvector` (tsvector) maintained by trigger; weighted A for title/titlealt, C for synopsis.
- **Partitioning**: `crawler.log`, `analytics.pageview`, `analytics.chaptersession` — RANGE by month. Drop old partitions instead of DELETE.
- **Denormalized counters**: `viewcount`, `followcount`, `chaptercount`, `ratingavg`, `ratingbayesian`, `threadcount`, `postcount` — maintained by application layer + background jobs.

### Execution order

```
00_SETUP  → 10_USERS → 20_CORE → 30_LIBRARY
→ 40_SOCIAL → 50_CRAWLER → 60_ANALYTICS → 70_SYSTEM
→ DATA/INITIAL_DATA
```

---

<!-- Template for future entries:

## [X.Y.Z] — YYYY-MM-DD

### Added
- `schema.tablename` — description

### Changed
- `schema.tablename.columnname` — old → new (reason)

### Removed
- `schema.tablename` — reason migration: `000X_...`

### Fixed
- description of non-breaking correction

-->
