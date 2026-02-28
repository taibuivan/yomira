# AGENTS.md

## Cursor Cloud specific instructions

### Project overview

Yomira is a self-hosted manga/comic reading platform. It is a **Go monolith** (single binary, single PostgreSQL instance) with the source code under `src/`. See `README.md` for the full product description and `CONTRIBUTING.md` for code style and PR conventions.

### Services

| Service | How to start | Default port |
|---|---|---|
| **PostgreSQL 16** | `sudo docker start postgres16` (or create: see below) | 5432 |
| **Redis 7** | `sudo docker start redis7` (or create: see below) | 6379 |
| **Yomira API** | `cd src && set -a && source .env && set +a && air -c .air.toml` | 8080 |

If containers don't exist yet:
```
sudo dockerd &>/tmp/dockerd.log &
sudo docker run -d --name postgres16 -e POSTGRES_USER=yomira -e POSTGRES_PASSWORD=password -e POSTGRES_DB=yomira -p 5432:5432 postgres:16
sudo docker run -d --name redis7 -p 6379:6379 redis:7
```

### Key gotchas

- **Environment variables**: The app reads from OS env vars (via `caarlos0/env`), **not** from `.env` files automatically. Either `source .env` before running the binary, or use `make dev` / `make run` (the Makefile includes and exports `.env`).
- **JWT keys required**: The app validates that `JWT_PRIVATE_KEY_PATH` and `JWT_PUBLIC_KEY_PATH` exist on disk at startup. Generate them in `src/keys/` if missing: `openssl genrsa -out src/keys/private.pem 2048 && openssl rsa -in src/keys/private.pem -pubout -out src/keys/public.pem`.
- **Migration directory must exist**: `src/data/migrations/` must contain at least one migration file. The migration runner (`migration.RunUp`) fails if the directory is empty.
- **Docker-in-Docker**: This cloud VM requires `fuse-overlayfs` storage driver and `iptables-legacy` for Docker to work. These are configured in `/etc/docker/daemon.json` and via `update-alternatives`.
- **golangci-lint revive config bug**: The `.golangci.yml` has a revive rule `increment-decrement# Use i++ not i += 1` with an inline comment that breaks the parser. This causes a non-fatal setup warning but lint still runs.
- **Go version**: `go.mod` requires Go 1.24. The VM's default Go may be older; ensure `/usr/local/go/bin` is in PATH after installing Go 1.24.

### Common commands

Standard commands are documented in `Makefile` (run `make help`). Key ones:

| Task | Command |
|---|---|
| Build | `make build` (from repo root) |
| Dev server (hot reload) | `make dev` (from repo root, or `cd src && air -c .air.toml`) |
| Lint | `make lint` (or `cd src && golangci-lint run --timeout 5m`) |
| Test | `make test` (or `cd src && go test -race -count=1 ./...`) |
| Format | `make fmt` |

### Working directory notes

- The `Makefile` runs from `/workspace` (repo root) and targets `./src/cmd/api`.
- The `.air.toml` and `.env` are in `/workspace/src/` — `air` must be run from there.
- `golangci-lint` config is at `/workspace/.golangci.yml`, but the Go module is in `/workspace/src/`, so run lint from `src/`.
