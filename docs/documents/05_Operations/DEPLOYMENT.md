# Deployment Guide — Yomira

> **Author:** tai.buivan.jp@gmail.com  
> **Version:** 1.0.0 — 2026-02-22

---

## Table of Contents

1. [Infrastructure Overview](#1-infrastructure-overview)
2. [CI/CD Pipeline](#2-cicd-pipeline)
3. [Environments](#3-environments)
4. [Build Process](#4-build-process)
5. [Database Migrations in Production](#5-database-migrations-in-production)
6. [Zero-Downtime Deployment](#6-zero-downtime-deployment)
7. [Rollback Procedure](#7-rollback-procedure)
8. [Secrets Management](#8-secrets-management)
9. [Post-Deployment Checklist](#9-post-deployment-checklist)

---

## 1. Infrastructure Overview

```
                        ┌─────────────────────────────────────┐
                        │         Cloudflare (CDN/WAF)        │
                        └────────────────┬────────────────────┘
                                         │
                        ┌────────────────▼────────────────────┐
                        │         Load Balancer (L7)          │
                        └────────┬────────────────────────────┘
                                 │
              ┌──────────────────┼───────────────────┐
              │                  │                   │
    ┌─────────▼───────┐ ┌───────▼────────┐ ┌───────▼────────┐
    │  Go API (pod 1) │ │ Go API (pod 2) │ │ Go API (pod N) │
    └─────────┬───────┘ └───────┬────────┘ └───────┬────────┘
              └──────────────────┼───────────────────┘
                                 │
              ┌──────────────────┼───────────────────┐
              │                  │                   │
    ┌─────────▼───────┐ ┌───────▼────────┐ ┌───────▼────────┐
    │  PostgreSQL 16  │ │  Redis 7       │ │ Cloudflare R2  │
    │  (primary +     │ │  (cluster)     │ │  (object store)│
    │   read replica) │ └────────────────┘ └────────────────┘
    └─────────────────┘
```

**Hosting options:**
| Service | Recommended | Alternative |
|---|---|---|
| Go API | Fly.io / Railway / GCP Cloud Run | Any container host |
| PostgreSQL | Supabase / Railway / RDS | Self-hosted |
| Redis | Upstash Redis | ElastiCache |
| Object Storage | Cloudflare R2 | AWS S3 |
| CDN | Cloudflare | CloudFront |

---

## 2. CI/CD Pipeline

### GitHub Actions — `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: yomira_test
          POSTGRES_USER: yomira
          POSTGRES_PASSWORD: test
        ports: ["5432:5432"]
        options: --health-cmd pg_isready --health-interval 5s

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true

      - name: Install tools
        run: |
          go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: Run migrations
        run: migrate -database "$DATABASE_URL" -path src/common/DML/migrations up
        env:
          DATABASE_URL: postgres://yomira:test@localhost:5432/yomira_test?sslmode=disable

      - name: Lint
        run: golangci-lint run ./...

      - name: Test
        run: go test -race -coverprofile=coverage.out ./...
        env:
          DATABASE_URL: postgres://yomira:test@localhost:5432/yomira_test?sslmode=disable

      - name: Security audit
        run: govulncheck ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: coverage.out
```

### GitHub Actions — `.github/workflows/deploy.yml`

```yaml
name: Deploy

on:
  push:
    branches: [main]   # Deploy on merge to main

jobs:
  deploy:
    runs-on: ubuntu-latest
    needs: [test]      # Only deploy if tests pass
    environment: production

    steps:
      - uses: actions/checkout@v4

      - name: Build Docker image
        run: |
          docker build -t yomira-api:${GITHUB_SHA} .
          docker tag yomira-api:${GITHUB_SHA} yomira-api:latest

      - name: Push to registry
        run: |
          echo "${{ secrets.REGISTRY_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin
          docker push ghcr.io/your-org/yomira-api:${GITHUB_SHA}

      - name: Run migrations on production DB
        run: |
          migrate -database "${{ secrets.PROD_DATABASE_URL }}" \
            -path src/common/DML/migrations up
        # Migrations run BEFORE new pods start — safe for additive changes

      - name: Deploy to Fly.io
        run: flyctl deploy --image ghcr.io/your-org/yomira-api:${GITHUB_SHA}
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}

      - name: Verify deployment
        run: |
          sleep 10
          curl --fail https://api.yomira.app/health/ready || exit 1
```

---

## 3. Environments

| Env | Branch | URL | DB | Auto-deploy |
|---|---|---|---|---|
| `development` | any | `localhost:8080` | Local Docker | Manual |
| `staging` | `develop` | `api.staging.yomira.app` | Staging DB | On push to `develop` |
| `production` | `main` | `api.yomira.app` | Production DB | On merge to `main` |

### Environment-specific config

```bash
# staging: same as production but with extended logging
ENV=staging
LOG_LEVEL=debug
LOG_FORMAT=json
CORS_ORIGINS=https://staging.yomira.app

# production
ENV=production
LOG_LEVEL=info
LOG_FORMAT=json
CORS_ORIGINS=https://yomira.app,https://www.yomira.app,https://admin.yomira.app
```

---

## 4. Build Process

### Dockerfile

```dockerfile
# Multi-stage build — production image ~20MB
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
    -o /bin/yomira ./cmd/server

# Final image
FROM scratch
COPY --from=builder /bin/yomira /yomira
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080
ENTRYPOINT ["/yomira"]
```

### Build locally

```bash
# Standard build
go build -o bin/yomira ./cmd/server

# Production build (optimized, with version)
CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always)" \
    -o bin/yomira ./cmd/server

# Docker build
docker build -t yomira-api:local .
docker run --env-file .env -p 8080:8080 yomira-api:local
```

---

## 5. Database Migrations in Production

### Strategy

Migrations run **before** new server pods start. This works because Yomira follows strict migration rules:
- Only additive changes (new columns nullable, new tables) take effect before code expects them
- Breaking changes require two deploys: (1) add + keep backward compat, (2) remove old

```bash
# In CI/CD pipeline — migration runs first
migrate -database "${PROD_DATABASE_URL}" -path src/common/DML/migrations up

# Then deploy new pods
flyctl deploy ...
```

### Manual migration (emergency)

```bash
# Connect to production DB and run migration manually
migrate -database "${PROD_DATABASE_URL}" -path src/common/DML/migrations up 1

# Check version
migrate -database "${PROD_DATABASE_URL}" -path src/common/DML/migrations version

# Rollback
migrate -database "${PROD_DATABASE_URL}" -path src/common/DML/migrations down 1
```

---

## 6. Zero-Downtime Deployment

### Rolling deployment (Fly.io)

```toml
# fly.toml
[deploy]
  strategy = "rolling"   # Replace instances one at a time

[[vm]]
  count = 2              # minimum 2 instances for zero-downtime
  memory = "512mb"
  cpus = 1
```

### Health check during deployment

New instances only receive traffic once they pass the health check:

```toml
[[services.http_checks]]
  path     = "/health/ready"
  interval = "5s"
  timeout  = "3s"
  grace_period = "10s"   # allow time for DB connection pool to warm up
```

### Graceful shutdown

```go
// cmd/server/main.go
func main() {
    srv := &http.Server{Addr: ":" + port, Handler: router}

    // Start server
    go func() { srv.ListenAndServe() }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // Graceful shutdown: wait for in-flight requests (30s max)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}
```

---

## 7. Rollback Procedure

### Automatic rollback (CI/CD)

If `curl --fail /health/ready` returns non-200 after deploy, GitHub Actions fails and Fly.io rolls back to previous version automatically.

### Manual rollback

```bash
# Fly.io: roll back to previous release
flyctl releases list              # find previous release number
flyctl deploy --image ghcr.io/your-org/yomira-api:{previous-sha}

# Roll back migration if needed (ONLY if new code is fully stopped)
migrate -database "${PROD_DATABASE_URL}" -path src/common/DML/migrations down 1

# Verify
curl https://api.yomira.app/health/ready
```

---

## 8. Secrets Management

### GitHub Secrets (for CI/CD)

| Secret | Used in |
|---|---|
| `PROD_DATABASE_URL` | Production DB connection |
| `STAGING_DATABASE_URL` | Staging DB connection |
| `FLY_API_TOKEN` | Fly.io deployment |
| `REGISTRY_TOKEN` | GitHub Container Registry push |

### Production secrets (Fly.io)

```bash
# Set secrets on Fly.io (stored encrypted, injected as env vars)
flyctl secrets set \
  DATABASE_URL="postgres://..." \
  REDIS_URL="redis://..." \
  JWT_PRIVATE_KEY="$(cat secrets/jwt_private.pem)" \
  JWT_PUBLIC_KEY="$(cat secrets/jwt_public.pem)" \
  RESEND_API_KEY="re_..." \
  S3_ACCESS_KEY="..." \
  S3_SECRET_KEY="..."

# List set secrets (values hidden)
flyctl secrets list
```

---

## 9. Post-Deployment Checklist

```
□ /health/ready returns 200 with all checks "ok"
□ /api/v1/.well-known/jwks.json returns valid JWKS
□ Migration version matches expected (migrate version)
□ Smoke test: POST /auth/login with test account
□ Check error rate in logs (should be < 0.1%)
□ Check p95 latency on /comics (should be < 200ms)
□ Verify CDN can reach media URLs (cdn.yomira.app/...)
□ Verify email delivery (send test email via admin panel)
□ Check Redis memory usage (should be < 80%)
□ Check DB connection pool usage
```
