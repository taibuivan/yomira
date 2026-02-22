// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package migration provides a thin wrapper around golang-migrate for
// running database schema migrations.
//
// # Architecture
//
// This package belongs to the Infrastructure layer. It enforces schema
// idempotency during application startup, ensuring the database is always
// in the correct state before traffic is served.
package migration

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	// pgx5 driver registers "pgx5" scheme for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	// file source reads .sql files from disk.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunUp applies all pending UP migrations.
//
// # Parameters
//   - dsn: A libpq-compatible DSN or postgres:// URL.
//   - migrationsPath: Filesystem path to the migrations directory.
//   - logger: Structured logger for migration events.
func RunUp(dsn string, migrationsPath string, logger *slog.Logger) error {
	// golang-migrate pgx/v5 driver expects "pgx5://" scheme.
	databaseURL := convertToPgx5DSN(dsn)
	sourceURL := "file://" + migrationsPath

	migrator, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("migration: failed to initialize: %w", err)
	}
	defer func() {
		sourceError, dbError := migrator.Close()
		if sourceError != nil {
			logger.Error("migration_source_close_failed", slog.Any("error", sourceError))
		}
		if dbError != nil {
			logger.Error("migration_db_close_failed", slog.Any("error", dbError))
		}
	}()

	// Enable verbose logging via the slog bridge.
	migrator.Log = &migrateLogger{logger: logger}

	currentVersion, isDirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("migration: failed to get current version: %w", err)
	}

	if isDirty {
		return fmt.Errorf("migration: database is in a dirty state at version %d (manual intervention required)", currentVersion)
	}

	logger.Info("migration_started", slog.Int("current_version", int(currentVersion)))

	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("migration_already_up_to_date")
			return nil
		}
		return fmt.Errorf("migration: up failed: %w", err)
	}

	newVersion, _, _ := migrator.Version()
	logger.Info("migration_successful",
		slog.Int("from_version", int(currentVersion)),
		slog.Int("to_version", int(newVersion)),
	)

	return nil
}

// convertToPgx5DSN ensures the DSN uses the pgx5:// scheme required by golang-migrate/v4.
func convertToPgx5DSN(dsn string) string {
	const pgPrefix = "postgres://"
	const pgqlPrefix = "postgresql://"
	const pgx5Prefix = "pgx5://"

	if len(dsn) >= len(pgx5Prefix) && dsn[:len(pgx5Prefix)] == pgx5Prefix {
		return dsn
	}

	if len(dsn) >= len(pgPrefix) && dsn[:len(pgPrefix)] == pgPrefix {
		return pgx5Prefix + dsn[len(pgPrefix):]
	}

	if len(dsn) >= len(pgqlPrefix) && dsn[:len(pgqlPrefix)] == pgqlPrefix {
		return pgx5Prefix + dsn[len(pgqlPrefix):]
	}

	return dsn
}

// migrateLogger adapts golang-migrate's logger interface to slog.
type migrateLogger struct {
	logger  *slog.Logger
	verbose bool
}

// Printf implements migrate.Logger.
func (l *migrateLogger) Printf(format string, args ...any) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// Verbose implements migrate.Logger.
func (l *migrateLogger) Verbose() bool {
	return l.verbose
}
