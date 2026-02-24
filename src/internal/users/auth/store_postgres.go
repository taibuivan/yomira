// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package postgres implements the storage layer for the Yomira application using PostgreSQL.
//
// # Architecture
//
// Repositories in this package are strictly separated from domain logic. They
// implement domain-defined interfaces (e.g., [UserRepository]) using the
// [pgxpool.Pool] connection manager.
//
// # err Mapping
//
// Storage-specific errors (like pgx.ErrNoRows) are mapped to domain-friendly
// [apperr.AppError] types to avoid leaking storage implementation details.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taibuivan/yomira/internal/platform/apperr"
)

// # User Repository

// PostgresUserRepository implements the UserRepository interface using pgx.
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL implementation of the UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

/*
Create persists a new user record into the users.account table.

Description: Deep-persists account metadata, ensuring timestamps are initialized
if not provided.

Parameters:
  - context: context.Context
  - user: *User (Entity to persist)

Returns:
  - error: Database constraint violations or connectivity errors
*/
func (repository *PostgresUserRepository) Create(context context.Context, user *User) error {
	const query = `
		INSERT INTO users.account (
			id, username, email, passwordhash, displayname, role, isverified, createdat, updatedat
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	_, err := repository.pool.Exec(context, query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.Role,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("postgres_user_repo_create_failed: %w", err)
	}

	return nil
}

/*
FindByEmail retrieves a user record by their unique email address.

Description: Performs a lookup on the account table, filtering out soft-deleted users.

Parameters:
  - context: context.Context
  - email: string

Returns:
  - *User: Hydrated account entity
  - error: apperr.NotFound or database errors
*/
func (repository *PostgresUserRepository) FindByEmail(context context.Context, email string) (*User, error) {
	const query = `
		SELECT id, username, email, passwordhash, displayname, role, isverified, createdat, updatedat
		FROM users.account
		WHERE email = $1 AND deletedat IS NULL`

	user := &User{}
	err := repository.pool.QueryRow(context, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("User not found with this email")
		}
		return nil, fmt.Errorf("postgres_user_repo_find_by_email_failed: %w", err)
	}

	return user, nil
}

/*
FindByUsername retrieves a user record by their unique username.

Description: Standard lookup by username for authentication and profile resolution.

Parameters:
  - context: context.Context
  - username: string

Returns:
  - *User: Hydrated account entity
  - error: apperr.NotFound or database errors
*/
func (repository *PostgresUserRepository) FindByUsername(context context.Context, username string) (*User, error) {
	const query = `
		SELECT id, username, email, passwordhash, displayname, role, isverified, createdat, updatedat
		FROM users.account
		WHERE username = $1 AND deletedat IS NULL`

	user := &User{}
	err := repository.pool.QueryRow(context, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("User not found with this username")
		}
		return nil, fmt.Errorf("postgres_user_repo_find_by_username_failed: %w", err)
	}

	return user, nil
}

/*
FindByID retrieves a user record by their unique ID.

Description: Primary key resolution for user accounts.

Parameters:
  - context: context.Context
  - id: string (UUIDv7)

Returns:
  - *User: Hydrated account entity
  - error: Not found or execution errors
*/
func (repository *PostgresUserRepository) FindByID(context context.Context, id string) (*User, error) {
	const query = `
		SELECT id, username, email, passwordhash, displayname, role, isverified, createdat, updatedat
		FROM users.account
		WHERE id = $1 AND deletedat IS NULL`

	user := &User{}
	err := repository.pool.QueryRow(context, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("User not found")
		}
		return nil, fmt.Errorf("postgres_user_repo_find_by_id_failed: %w", err)
	}

	return user, nil
}

/*
Update persists changes to a user's mutable profile fields.

Description: Synchronizes the in-memory user state with the database,
refreshing the updatedat timestamp.

Parameters:
  - context: context.Context
  - user: *User

Returns:
  - error: Update failures
*/
func (repository *PostgresUserRepository) Update(context context.Context, user *User) error {
	const query = `
		UPDATE users.account
		SET username = $2, displayname = $3, avatarurl = $4, bio = $5, updatedat = $6
		WHERE id = $1 AND deletedat IS NULL`

	user.UpdatedAt = time.Now()
	_, err := repository.pool.Exec(context, query,
		user.ID,
		user.Username,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("postgres_user_repo_update_failed: %w", err)
	}

	return nil
}

/*
UpdatePassword updates only the password hash for a specific user.

Parameters:
  - context: context.Context
  - userID: string
  - newHash: string

Returns:
  - error: Execution errors
*/
func (repository *PostgresUserRepository) UpdatePassword(context context.Context, userID, newHash string) error {
	const query = `
		UPDATE users.account
		SET passwordhash = $2, updatedat = $3
		WHERE id = $1 AND deletedat IS NULL`

	_, err := repository.pool.Exec(context, query, userID, newHash, time.Now())
	if err != nil {
		return fmt.Errorf("postgres_user_repo_update_password_failed: %w", err)
	}

	return nil
}

/*
SoftDelete marks a user account as deleted using their ID.

Description: Retention-friendly deletion by setting deletedat.

Parameters:
  - context: context.Context
  - id: string

Returns:
  - error: Side-effect failures
*/
func (repository *PostgresUserRepository) SoftDelete(context context.Context, id string) error {
	const query = "UPDATE users.account SET deletedat = $2 WHERE id = $1"
	_, err := repository.pool.Exec(context, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("postgres_user_repo_soft_delete_failed: %w", err)
	}
	return nil
}

/*
MarkVerified updates the user's status to isverified = true.

Description: Post-verification cleanup to activate the account.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - error: Database errors
*/
func (repository *PostgresUserRepository) MarkVerified(context context.Context, userID string) error {
	const query = "UPDATE users.account SET isverified = TRUE, updatedat = $2 WHERE id = $1"
	_, err := repository.pool.Exec(context, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("postgres_user_repo_mark_verified_failed: %w", err)
	}
	return nil
}

// # Session Repository

// PostgresSessionRepository implements the SessionRepository interface.
type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a new PostgreSQL implementation of SessionRepository.
func NewSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

/*
Create persists a new session record into the users.session table.

Description: Records a successful authentication session in persistent storage.

Parameters:
  - context: context.Context
  - session: *Session

Returns:
  - error: Storage failures
*/
func (repository *PostgresSessionRepository) Create(context context.Context, session *Session) error {
	const query = `
		INSERT INTO users.session (
			id, userid, tokenhash, useragent, ipaddress, expiresat, isrevoked, createdat
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	_, err := repository.pool.Exec(context, query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.IsRevoked,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("postgres_session_repo_create_failed: %w", err)
	}

	return nil
}

/*
FindByTokenHash retrieves an active session by its unique token hash.

Description: Securely resolves a refresh token hash into an active session.

Parameters:
  - context: context.Context
  - tokenHash: string

Returns:
  - *Session: Hydrated session metadata
  - error: apperr.NotFound or execution errors
*/
func (repository *PostgresSessionRepository) FindByTokenHash(context context.Context, tokenHash string) (*Session, error) {
	const query = `
		SELECT id, userid, tokenhash, useragent, ipaddress, expiresat, isrevoked, createdat
		FROM users.session
		WHERE tokenhash = $1 AND isrevoked = FALSE AND expiresat > NOW()`

	session := &Session{}
	err := repository.pool.QueryRow(context, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.IsRevoked,
		&session.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("Session not found or expired")
		}
		return nil, fmt.Errorf("postgres_session_repo_find_failed: %w", err)
	}

	return session, nil
}

/*
Revoke marks a specific session as revoked.

Parameters:
  - context: context.Context
  - sessionID: string

Returns:
  - error: Revocation failures
*/
func (repository *PostgresSessionRepository) Revoke(context context.Context, sessionID string) error {
	const query = "UPDATE users.session SET isrevoked = TRUE WHERE id = $1"
	_, err := repository.pool.Exec(context, query, sessionID)
	if err != nil {
		return fmt.Errorf("postgres_session_repo_revoke_failed: %w", err)
	}
	return nil
}

/*
RevokeAll marks all active sessions for a user as revoked.

Description: Security nuking of all active sessions for a user.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - error: Batch revocation failures
*/
func (repository *PostgresSessionRepository) RevokeAll(context context.Context, userID string) error {
	const query = "UPDATE users.session SET isrevoked = TRUE WHERE userid = $1 AND isrevoked = FALSE"
	_, err := repository.pool.Exec(context, query, userID)
	if err != nil {
		return fmt.Errorf("postgres_session_repo_revoke_all_failed: %w", err)
	}
	return nil
}

/*
RevokeOthers marks all active sessions for a user as revoked, except for one.

Parameters:
  - context: context.Context
  - userID: string
  - currentSessionID: string

Returns:
  - error: Filtered revocation failures
*/
func (repository *PostgresSessionRepository) RevokeOthers(context context.Context, userID, currentSessionID string) error {
	const query = "UPDATE users.session SET isrevoked = TRUE WHERE userid = $1 AND id != $2 AND isrevoked = FALSE"
	_, err := repository.pool.Exec(context, query, userID, currentSessionID)
	if err != nil {
		return fmt.Errorf("postgres_session_repo_revoke_others_failed: %w", err)
	}
	return nil
}

/*
DeleteExpired permanently removes all sessions that have passed their expiration.

Description: Cleanup task to reclaim storage from stale sessions.

Parameters:
  - context: context.Context

Returns:
  - error: Cleanup failures
*/
func (repository *PostgresSessionRepository) DeleteExpired(context context.Context) error {
	const query = "DELETE FROM users.session WHERE expiresat <= NOW()"
	_, err := repository.pool.Exec(context, query)
	if err != nil {
		return fmt.Errorf("postgres_session_repo_delete_expired_failed: %w", err)
	}
	return nil
}
