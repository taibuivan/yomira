// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package account (Postgres) implements the storage layer for user meta-data.

It provides optimized PostgreSQL implementations for managing user profiles,
mapping reading preferences, and auditing active sessions.

# Schema Table Mapping
  - users.account: Master identity and profile data.
  - users.readingpreference: 1:1 user settings configuration.
  - users.session: Active device sessions and security metadata.
*/
package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
	"github.com/taibuivan/yomira/internal/users/auth"
)

// # Repository Implementations

// PostgresAccountRepository implements [AccountRepository] using pgx.
type PostgresAccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates a new Postgres implementation for profile management.
func NewAccountRepository(pool *pgxpool.Pool) *PostgresAccountRepository {
	return &PostgresAccountRepository{pool: pool}
}

// PostgresPreferencesRepository implements [PreferencesRepository] using pgx.
type PostgresPreferencesRepository struct {
	pool *pgxpool.Pool
}

// NewPreferencesRepository creates a new Postgres implementation for user settings.
func NewPreferencesRepository(pool *pgxpool.Pool) *PostgresPreferencesRepository {
	return &PostgresPreferencesRepository{pool: pool}
}

// PostgresSessionRepository implements [SessionRepository] using pgx.
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a new Postgres implementation for session auditing.
func NewSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

// # AccountRepository Methods

/*
FindByID retrieves a user record from the users.account table.

Parameters:
  - context: context.Context
  - id: string (UUID)

Returns:
  - *auth.User: Hydrated identity entity
  - error: apperr.NotFound or database execution failure
*/
func (repository *PostgresAccountRepository) FindByID(context context.Context, id string) (*auth.User, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.ID, schema.UserAccount.Username, schema.UserAccount.Email,
		schema.UserAccount.Password, schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL,
		schema.UserAccount.Bio, schema.UserAccount.Website, schema.UserAccount.Role,
		schema.UserAccount.IsVerified, schema.UserAccount.CreatedAt, schema.UserAccount.UpdatedAt,
		schema.UserAccount.Table, schema.UserAccount.ID, schema.UserAccount.DeletedAt,
	)

	user := &auth.User{}
	err := repository.pool.QueryRow(context, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Bio,
		&user.Website,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("Account")
		}
		return nil, fmt.Errorf("postgres_account_repo_find_by_id_failed: %w", err)
	}

	return user, nil
}

/*
Update modifies the mutable profile metadata of a user.

Description: This method specifically syncs the DisplayName, AvatarURL, Bio,
and Website fields, while refreshing the updatedat timestamp.

Parameters:
  - context: context.Context
  - user: *auth.User

Returns:
  - error: Update failures
*/
func (repository *PostgresAccountRepository) Update(context context.Context, user *auth.User) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3, %s = $4, %s = $5, %s = $6
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.Table,
		schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL, schema.UserAccount.Bio,
		schema.UserAccount.Website, schema.UserAccount.UpdatedAt,
		schema.UserAccount.ID, schema.UserAccount.DeletedAt,
	)

	_, err := repository.pool.Exec(context, query,
		user.ID,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		user.Website,
		time.Now(),
	)

	// If the update fails, return an error
	if err != nil {
		return fmt.Errorf("postgres_account_repo_update_failed: %w", err)
	}

	return nil
}

/*
SoftDelete flags a user account as logically destroyed.

Parameters:
  - context: context.Context
  - id: string

Returns:
  - error: Execution failures
*/
func (repository *PostgresAccountRepository) SoftDelete(context context.Context, id string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1`,
		schema.UserAccount.Table, schema.UserAccount.DeletedAt, schema.UserAccount.ID)
	_, err := repository.pool.Exec(context, query, id)
	return err
}

// # PreferencesRepository Methods

/*
FindByUserID retrieves the serialized reading settings for a specific user.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - *Preferences: Hydrated setting entity
  - error: apperr.NotFound or retrieval failures
*/
func (repository *PostgresPreferencesRepository) FindByUserID(context context.Context, userID string) (*Preferences, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1`,
		schema.UserPreferences.UserID, schema.UserPreferences.ReadingMode, schema.UserPreferences.PageFit,
		schema.UserPreferences.PreloadPages, schema.UserPreferences.HideNSFW, schema.UserPreferences.HideLanguages,
		schema.UserPreferences.DataSaver,
		schema.UserPreferences.Table,
		schema.UserPreferences.UserID,
	)

	prefs := &Preferences{}
	err := repository.pool.QueryRow(context, query, userID).Scan(
		&prefs.UserID,
		&prefs.ReadingMode,
		&prefs.PageFit,
		&prefs.PreloadPages,
		&prefs.HideNSFW,
		&prefs.HideLanguages,
		&prefs.DataSaver,
	)

	// If the query fails, return an error
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("Preferences")
		}
		return nil, fmt.Errorf("postgres_preference_repo_find_failed: %w", err)
	}

	return prefs, nil
}

/*
Upsert saves a user's preferences using an ON CONFLICT UPDATE strategy.

Parameters:
  - context: context.Context
  - prefs: *Preferences

Returns:
  - error: Synchronization failures
*/
func (repository *PostgresPreferencesRepository) Upsert(context context.Context, prefs *Preferences) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, %s, %s
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (%s) DO UPDATE SET
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s`,
		schema.UserPreferences.Table,
		schema.UserPreferences.UserID, schema.UserPreferences.ReadingMode, schema.UserPreferences.PageFit,
		schema.UserPreferences.PreloadPages, schema.UserPreferences.HideNSFW, schema.UserPreferences.HideLanguages,
		schema.UserPreferences.DataSaver,
		schema.UserPreferences.UserID,
		schema.UserPreferences.ReadingMode, schema.UserPreferences.ReadingMode,
		schema.UserPreferences.PageFit, schema.UserPreferences.PageFit,
		schema.UserPreferences.PreloadPages, schema.UserPreferences.PreloadPages,
		schema.UserPreferences.HideNSFW, schema.UserPreferences.HideNSFW,
		schema.UserPreferences.HideLanguages, schema.UserPreferences.HideLanguages,
		schema.UserPreferences.DataSaver, schema.UserPreferences.DataSaver,
	)

	_, err := repository.pool.Exec(context, query,
		prefs.UserID,
		prefs.ReadingMode,
		prefs.PageFit,
		prefs.PreloadPages,
		prefs.HideNSFW,
		prefs.HideLanguages,
		prefs.DataSaver,
	)

	// If the upsert fails, return an error
	if err != nil {
		return fmt.Errorf("postgres_preference_repo_upsert_failed: %w", err)
	}

	return nil
}

// # SessionRepository Methods

/*
FindActiveByUserID retrieves all valid device sessions for a user.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - []SessionInfo: Collection of active devices
  - error: Database retrieval failures
*/
func (repository *PostgresSessionRepository) FindActiveByUserID(context context.Context, userID string) ([]SessionInfo, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL AND %s > NOW()
		ORDER BY %s DESC`,
		schema.UserSession.ID, schema.UserSession.DeviceName, schema.UserSession.IPAddress,
		schema.UserSession.CreatedAt, schema.UserSession.ExpiresAt,
		schema.UserSession.Table,
		schema.UserSession.UserID, schema.UserSession.RevokedAt, schema.UserSession.ExpiresAt,
		schema.UserSession.CreatedAt,
	)

	rows, err := repository.pool.Query(context, query, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres_session_repo_list_active_failed: %w", err)
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var sess SessionInfo
		var ip interface{}
		if err := rows.Scan(&sess.ID, &sess.DeviceName, &ip, &sess.CreatedAt, &sess.ExpiresAt); err != nil {
			return nil, err
		}
		if ip != nil {
			sess.IPAddress = fmt.Sprintf("%v", ip)
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

/*
Revoke marks a single session as permanently revoked.

Parameters:
  - context: context.Context
  - userID: string (Security: validation of ownership)
  - sessionID: string

Returns:
  - error: Update failures
*/
func (repository *PostgresSessionRepository) Revoke(context context.Context, userID, sessionID string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s = $2`,
		schema.UserSession.Table, schema.UserSession.RevokedAt, schema.UserSession.ID, schema.UserSession.UserID)
	_, err := repository.pool.Exec(context, query, sessionID, userID)
	return err
}

/*
RevokeOthers marks all sessions except the current one as revoked.

Parameters:
  - context: context.Context
  - userID: string
  - currentSessionID: string

Returns:
  - error: Batch update failures
*/
func (repository *PostgresSessionRepository) RevokeOthers(context context.Context, userID, currentSessionID string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s != $2 AND %s IS NULL`,
		schema.UserSession.Table, schema.UserSession.RevokedAt, schema.UserSession.UserID,
		schema.UserSession.ID, schema.UserSession.RevokedAt)
	_, err := repository.pool.Exec(context, query, userID, currentSessionID)
	return err
}

/*
RevokeAll terminates every session for a user.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - error: Batch update failures
*/
func (repository *PostgresSessionRepository) RevokeAll(context context.Context, userID string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = NOW() WHERE %s = $1 AND %s IS NULL`,
		schema.UserSession.Table, schema.UserSession.RevokedAt, schema.UserSession.UserID, schema.UserSession.RevokedAt)
	_, err := repository.pool.Exec(context, query, userID)
	return err
}
