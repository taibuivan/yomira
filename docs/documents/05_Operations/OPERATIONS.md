# Operations Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Local Development Setup](#1-local-development-setup)
2. [Environment Variables](#2-environment-variables)
3. [Docker Compose](#3-docker-compose)
4. [Running the Server](#4-running-the-server)
5. [Health Check Endpoints](#5-health-check-endpoints)
6. [Database Management](#6-database-management)
7. [Redis Management](#7-redis-management)
8. [Monthly Maintenance Tasks](#8-monthly-maintenance-tasks)

---

## 1. Local Development Setup

### Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | ≥ 1.22 | https://go.dev/dl/ |
| Docker Desktop | Latest | https://docker.com |
| golang-migrate CLI | Latest | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |
| `psql` client | ≥ 16 | bundled with PostgreSQL |
| `air` (hot reload) | Latest | `go install github.com/air-verse/air@latest` |

### First-time setup

```bash
# 1. Clone and enter project
git clone https://github.com/your-org/yomira.git && cd yomira

# 2. Copy environment file and fill in secrets
cp .env.example .env

# 3. Start infrastructure
docker compose up -d postgres redis minio mailhog

# 4. Wait for PostgreSQL
docker compose exec postgres pg_isready -U yomira

# 5. Run migrations
migrate -database "${DATABASE_URL}" -path src/common/DML/migrations up

# 6. Seed initial data
psql "${DATABASE_URL}" -f src/common/DML/DATA/INITIAL_DATA.sql

# 7. Start the server with hot reload
air
```

---

## 2. Environment Variables

### Required (server will not start without these)

| Variable | Example | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://yomira:pass@localhost:5432/yomira?sslmode=disable` | PostgreSQL DSN |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis URL |
| `JWT_PRIVATE_KEY_PATH` | `./secrets/jwt_private.pem` | RS256 private key for signing |
| `JWT_PUBLIC_KEY_PATH` | `./secrets/jwt_public.pem` | RS256 public key for verification |
| `APP_URL` | `http://localhost:3000` | Frontend base URL (used in email links) |
| `UNSUBSCRIBE_SECRET` | `changeme32byteshexstring...` | HMAC secret for unsubscribe tokens |

### Optional (have defaults)

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `ENV` | `development` | `development` \| `staging` \| `production` |
| `LOG_LEVEL` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `LOG_FORMAT` | `text` | `text` (dev) \| `json` (prod) |
| `SESSION_TTL_DAYS` | `30` | Refresh token lifetime in days |
| `CORS_ORIGINS` | `http://localhost:3000,http://localhost:5173` | Allowed CORS origins |

### Mail

| Variable | Default | Description |
|---|---|---|
| `MAIL_PROVIDER` | `smtp` | `resend` \| `sendgrid` \| `smtp` |
| `MAIL_FROM` | `noreply@yomira.app` | Default sender |
| `SMTP_HOST` | `localhost` | Dev: use Mailhog |
| `SMTP_PORT` | `1025` | |
| `RESEND_API_KEY` | — | Required if `MAIL_PROVIDER=resend` |
| `SENDGRID_API_KEY` | — | Required if `MAIL_PROVIDER=sendgrid` |

### Object Storage

| Variable | Example | Description |
|---|---|---|
| `S3_BUCKET` | `yomira-media` | Bucket name |
| `S3_ENDPOINT` | `http://localhost:9000` | R2/S3/MinIO endpoint |
| `S3_REGION` | `auto` | R2:`auto`; AWS: `ap-southeast-1` |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | — | Credentials |
| `CDN_BASE_URL` | `http://localhost:9000/yomira-media` | Public media URL prefix |
| `PRESIGN_TTL` | `900` | Presigned URL TTL (seconds) |

### OAuth (optional per provider)

| Variable | Description |
|---|---|
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth2 |
| `DISCORD_CLIENT_ID` / `DISCORD_CLIENT_SECRET` | Discord OAuth2 |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth2 |

---

## 3. Docker Compose

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: yomira
      POSTGRES_PASSWORD: yomira_dev
      POSTGRES_DB: yomira
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "yomira"]
      interval: 5s; timeout: 3s; retries: 5

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --appendonly yes
    volumes: [redis_data:/data]

  minio:
    image: minio/minio
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes: [minio_data:/data]

  mailhog:
    image: mailhog/mailhog
    ports: ["1025:1025", "8025:8025"]   # SMTP + Web UI

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### Common commands

```bash
docker compose up -d                          # start all
docker compose up -d postgres redis           # infra only
docker compose logs -f postgres               # follow logs
docker compose exec postgres psql -U yomira   # psql shell
docker compose exec redis redis-cli           # redis CLI
docker compose down                           # stop
docker compose down -v                        # stop + delete volumes (DESTRUCTIVE)
```

---

## 4. Running the Server

```bash
air                          # hot reload (development)
go run ./cmd/server          # standard
go build -o bin/yomira ./cmd/server && ./bin/yomira  # production build
LOG_LEVEL=debug go run ./cmd/server                   # debug logging
```

---

## 5. Health Check Endpoints

| Endpoint | Description |
|---|---|
| `GET /health` | Liveness — always 200 if process is running |
| `GET /health/ready` | Readiness — checks DB + Redis connectivity |
| `GET /api/v1/.well-known/jwks.json` | JWT public key set |

```json
// GET /health/ready → 200
{ "status": "ready", "checks": { "database": "ok", "redis": "ok", "storage": "ok" } }

// GET /health/ready → 503 (dependency down)
{ "status": "not_ready", "checks": { "database": "ok", "redis": "error: connection refused", "storage": "ok" } }
```

---

## 6. Database Management

```bash
# Shell
docker compose exec postgres psql -U yomira -d yomira

# Table sizes
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname IN ('users','core','library','social','crawler','analytics','system')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

# Migration status
SELECT version, dirty FROM schema_migrations;

# Active connections
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

# Backup
pg_dump "${DATABASE_URL}" --format=custom --file=backup_$(date +%Y%m%d_%H%M%S).dump

# Restore
pg_restore --clean --if-exists --dbname "${DATABASE_URL}" backup_file.dump
```

---

## 7. Redis Management

```bash
docker compose exec redis redis-cli

KEYS ratelimit:*             # rate limit keys
TTL ratelimit:ip:1.2.3.4    # TTL of a key
DEL ratelimit:ip:1.2.3.4    # clear rate limit (dev)
KEYS search:*               # search cache keys
MONITOR                     # real-time command monitor
FLUSHDB                     # clear DB (dev only — DESTRUCTIVE)
```

---

## 8. Monthly Maintenance Tasks

| Task | Frequency | How |
|---|---|---|
| Create next month's analytics partition | 1st of month | `POST /admin/batch/analytics/create-partition` (auto-scheduled) |
| Recalculate Bayesian ratings | Weekly | `POST /admin/batch/core/recalc-ratings` |
| Vacuum + analyze hot tables | Weekly | `VACUUM ANALYZE core.comic; VACUUM ANALYZE social.comment;` |
| Prune expired sessions | Daily | Scheduled Go worker (automatic) |
| Anonymize old IPs (90d) | Daily | Scheduled Go worker (automatic) |
| Check disk usage | Weekly | `df -h` + table size query |
| Review slow queries | Weekly | `SELECT ... FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;` |
| Update Go dependencies | Monthly | `go get -u ./...; go mod tidy` |
| Rotate JWT keys | Annually | Generate new PEM, update env, rolling restart |

### Monitoring checklist

- [ ] PostgreSQL: slow query log (`log_min_duration_statement = 500`)
- [ ] Redis: memory usage (alert at 80%)
- [ ] `crawler.job` rows stuck in `running` > 30 min
- [ ] `users.session` row count growth (cleanup job running?)
- [ ] API error rate on `5xx` responses
- [ ] Disk usage on object storage
- [ ] `govulncheck` weekly security scan
