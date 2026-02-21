# Yomira — Documentation Index

> **Author:** tai.buivan.jp@gmail.com  
> **Updated:** 2026-02-22

This folder contains all design and implementation documents for the Yomira platform.

---

## Structure

```
docs/documents/
├── 01_Architecture/     System design, ADRs, deployment roadmap
├── 02_Backend/          Go server, API design, package conventions
├── 03_Database/         PostgreSQL schema, ERD, migrations, queries
├── 04_Frontend/         UI design system, templates (Phase 2)
└── 05_Operations/       DevOps, local setup, maintenance runbook
```

---

## Documents

| # | File | Description | Status |
|---|---|---|---|
| 01 | [ARCHITECTURE.md](01_Architecture/ARCHITECTURE.md) | System overview, tech stack, ADRs, deployment phases, data flow, security | ✅ Active |
| 02 | [BACKEND.md](02_Backend/BACKEND.md) | Go package philosophy, project structure, domain/service/storage/API layers | ✅ Active |
| 03 | [DATABASE.md](03_Database/DATABASE.md) | Schema map, changelog, index strategy, query patterns, migrations | ✅ Active |
| 03 | [ERD.md](03_Database/ERD.md) | Mermaid ER diagrams (GitHub-rendered), split by schema | ✅ Active |
| 04 | [FRONTEND.md](04_Frontend/FRONTEND.md) | Page inventory, design system, reader features | 🔲 Planned |
| 05 | [OPERATIONS.md](05_Operations/OPERATIONS.md) | Docker Compose, env vars, monthly maintenance, monitoring | 🔲 Planned |

### API Reference

| File | Schema / Scope | Endpoints | Status |
|---|---|---|---|
| [API_CONVENTIONS.md](02_Backend/api/API_CONVENTIONS.md) | Global | Auth, Response Envelope, Errors, Pagination, Rate Limits, CORS | ✅ v1.0.0 |
| [USERS_API.md](02_Backend/api/USERS_API.md) | `10_USERS` | Auth, Profile, Sessions, Follow, Preferences, Admin | ✅ v1.0.0 |
| [CORE_API.md](02_Backend/api/CORE_API.md) | `20_CORE` | Languages, Authors, Artists, Tags, Groups, Comics, Chapters, Pages | ✅ v1.0.0 |
| [LIBRARY_API.md](02_Backend/api/LIBRARY_API.md) | `30_LIBRARY` | Library Entry, Custom Lists, Reading Progress, Chapter Read, View History | ✅ v1.0.0 |
| [SOCIAL_API.md](02_Backend/api/SOCIAL_API.md) | `40_SOCIAL` | Ratings, Comments, Notifications, Recommendations, Feed, Forum, Reports | ✅ v1.0.0 |
| [CRAWLER_API.md](02_Backend/api/CRAWLER_API.md) | `50_CRAWLER` | Sources, Comic Sources, Jobs, Logs (Admin/Internal) | ✅ v1.0.0 |
| [ANALYTICS_API.md](02_Backend/api/ANALYTICS_API.md) | `60_ANALYTICS` | Page Views, Chapter Sessions, Dashboard Stats (Admin/Internal) | ✅ v1.0.0 |
| [SYSTEM_API.md](02_Backend/api/SYSTEM_API.md) | `70_SYSTEM` | Audit Log, Settings, Announcements | ✅ v1.0.0 |
| [BATCH_API.md](02_Backend/api/BATCH_API.md) | Cross-schema | Background Workers, Scheduled Jobs, Admin Batch Triggers | ✅ v1.0.0 |
| [MAIL_API.md](02_Backend/api/MAIL_API.md) | Go service | Email Verification, Password Reset, Email Change, Notifications, Admin Mail | ✅ v1.0.0 |
| [UPLOAD_API.md](02_Backend/api/UPLOAD_API.md) | Go service + S3 | Presigned Upload, Avatar, Covers, Art Gallery, Chapter Pages | ✅ v1.0.0 |
| [SEARCH_API.md](02_Backend/api/SEARCH_API.md) | Cross-schema | Global Search, Autocomplete, Comic/Chapter/User/Forum Search | ✅ v1.0.0 |


---

## Quick Links

- **DML source:** `src/common/DML/` — SQL schema files (the single source of truth)
- **ERD (rendered):** `docs/documents/03_Database/ERD.md` — view on GitHub
- **Changelog:** `src/common/DML/CHANGELOG.md` — schema version history
