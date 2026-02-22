# =============================================================================
# Yomira — Makefile
# Author: tai.buivan.jp@gmail.com
# =============================================================================

BINARY      := yomira
CMD         := ./src/cmd/api
MIGRATIONS  := ./src/common/DML/migrations
COVERAGE    := coverage.out
GO_VERSION  := 1.22

# Load .env if present (for local dev targets that need DATABASE_URL, etc.)
-include .env
export

.PHONY: help build run dev test test-unit test-integration coverage \
        lint fmt vet tidy migrate-up migrate-down migrate-status \
        generate mock-gen clean docker-up docker-down seed

# =============================================================================
# HELP
# =============================================================================
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# =============================================================================
# BUILD
# =============================================================================
build: ## Build the API binary
	@echo "→ Building $(BINARY)..."
	go build -o bin/$(BINARY) $(CMD)

build-release: ## Build optimised release binary (stripped, no debug info)
	@echo "→ Building release $(BINARY)..."
	go build -ldflags="-s -w" -trimpath -o bin/$(BINARY) $(CMD)

# =============================================================================
# RUN
# =============================================================================
run: build ## Build and run the API server
	./bin/$(BINARY)

dev: ## Run the API server with live-reload (requires air: go install github.com/air-verse/air@latest)
	air -c .air.toml

# =============================================================================
# TESTING
# =============================================================================
test: ## Run all tests with race detector
	go test -race -count=1 ./...

test-unit: ## Run unit tests only (no integration, no Docker)
	go test -race -short -count=1 ./...

test-integration: ## Run integration tests (requires Docker for testcontainers)
	go test -race -tags=integration -count=1 ./tests/integration/...

test-verbose: ## Run all tests with verbose output
	go test -v -race -count=1 ./...

test-run: ## Run a single test function (usage: make test-run TEST=TestComicService_GetByID)
	go test -v -race -run $(TEST) ./...

coverage: ## Generate and open HTML coverage report
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE)

coverage-summary: ## Show coverage summary by package
	go test -race -coverprofile=$(COVERAGE) -covermode=atomic ./... && \
	go tool cover -func=$(COVERAGE) | sort -k3 -n

# =============================================================================
# CODE QUALITY
# =============================================================================
lint: ## Run golangci-lint (must pass — zero new warnings policy)
	golangci-lint run --timeout 5m

lint-fix: ## Run golangci-lint with auto-fix where possible
	golangci-lint run --fix --timeout 5m

fmt: ## Format all Go code with gofmt + goimports
	gofmt -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

# =============================================================================
# MIGRATIONS
# =============================================================================
migrate-up: ## Apply all pending database migrations
	@echo "→ Running migrations UP..."
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS) up

migrate-down: ## Roll back the last migration
	@echo "→ Rolling back 1 migration..."
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS) down 1

migrate-down-all: ## Roll back ALL migrations (destructive — dev only)
	@echo "→ Rolling back ALL migrations..."
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS) down -all

migrate-status: ## Show current migration version
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS) version

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=add_comic_index)
	@[ "${NAME}" ] || ( echo "Usage: make migrate-create NAME=<migration_name>"; exit 1 )
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(NAME)

# =============================================================================
# CODE GENERATION
# =============================================================================
generate: ## Run go generate across the project
	go generate ./...

mock-gen: ## Regenerate all interface mocks using mockery
	mockery --all --dir internal/domain \
	        --output internal/domain/mocks \
	        --outpkg mocks \
	        --with-expecter

# =============================================================================
# DOCKER (local dev)
# =============================================================================
docker-up: ## Start PostgreSQL + Redis via docker-compose
	docker compose -f docker/compose.dev.yml up -d

docker-down: ## Stop and remove dev containers
	docker compose -f docker/compose.dev.yml down

docker-logs: ## Tail logs from dev containers
	docker compose -f docker/compose.dev.yml logs -f

seed: ## Seed the database with development fixtures
	go run ./src/cmd/seed

# =============================================================================
# SECURITY
# =============================================================================
audit: ## Run govulncheck for known vulnerabilities in dependencies
	govulncheck ./...

# =============================================================================
# CLEAN
# =============================================================================
clean: ## Remove build artifacts and coverage files
	rm -rf bin/ $(COVERAGE)
	go clean -testcache

# =============================================================================
# CI COMPOSITE TARGETS
# These are what the CI pipeline runs — run them locally before pushing.
# =============================================================================
ci: fmt-check vet lint test-unit ## Full CI check (fast — no integration tests)

ci-full: fmt-check vet lint test ## Full CI check including integration tests

fmt-check: ## Check formatting without modifying files (used in CI)
	@test -z "$$(gofmt -l . | grep -v vendor)" || \
	  (echo "gofmt: the following files are not formatted:" && gofmt -l . && exit 1)
