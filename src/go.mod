module github.com/taibuivan/yomira

go 1.24.0

require (
	// Environment variable parsing
	github.com/caarlos0/env/v11 v11.3.1
	// HTTP router
	github.com/go-chi/chi/v5 v5.2.1

	// UUID v7
	github.com/google/uuid v1.6.0

	// PostgreSQL driver + connection pool
	github.com/jackc/pgx/v5 v5.7.2

	// Unicode text normalization (used by pkg/slug)
	golang.org/x/text v0.31.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect

	// Password hashing
	golang.org/x/crypto v0.45.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/time v0.14.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
