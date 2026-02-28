// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package api implements the observability endpoints for the Yomira platform.

It provides standard Kubernetes-style probes (liveness, readiness) to monitor the
operational health of the application and its critical dependencies.

Architecture:

  - Liveness: Returns 200 OK as long as the process is running.
  - Readiness: Performs shallow pings of Postgres and Redis to verify connectivity.

These handlers ensure that traffic is only routed to "warm" instances that are
fully connected to the data plane.
*/
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/database/schema"
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
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		schema.UserAccount.Table,
		schema.UserAccount.ID, schema.UserAccount.Username, schema.UserAccount.Email,
		schema.UserAccount.Password, schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL,
		schema.UserAccount.Bio, schema.UserAccount.Website, schema.UserAccount.Role,
		schema.UserAccount.IsVerified, schema.UserAccount.CreatedAt, schema.UserAccount.UpdatedAt,
	)

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
		user.AvatarURL,
		user.Bio,
		user.Website,
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
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.ID, schema.UserAccount.Username, schema.UserAccount.Email,
		schema.UserAccount.Password, schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL,
		schema.UserAccount.Bio, schema.UserAccount.Website, schema.UserAccount.Role,
		schema.UserAccount.IsVerified, schema.UserAccount.CreatedAt, schema.UserAccount.UpdatedAt,
		schema.UserAccount.Table, schema.UserAccount.Email, schema.UserAccount.DeletedAt,
	)

	user := &User{}
	err := repository.pool.QueryRow(context, query, email).Scan(
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
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.ID, schema.UserAccount.Username, schema.UserAccount.Email,
		schema.UserAccount.Password, schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL,
		schema.UserAccount.Bio, schema.UserAccount.Website, schema.UserAccount.Role,
		schema.UserAccount.IsVerified, schema.UserAccount.CreatedAt, schema.UserAccount.UpdatedAt,
		schema.UserAccount.Table, schema.UserAccount.Username, schema.UserAccount.DeletedAt,
	)

	user := &User{}
	err := repository.pool.QueryRow(context, query, username).Scan(
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

	user := &User{}
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
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3, %s = $4, %s = $5, %s = $6, %s = $7
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.Table,
		schema.UserAccount.Username, schema.UserAccount.DisplayName, schema.UserAccount.AvatarURL,
		schema.UserAccount.Bio, schema.UserAccount.Website, schema.UserAccount.UpdatedAt,
		schema.UserAccount.ID, schema.UserAccount.DeletedAt,
	)

	user.UpdatedAt = time.Now()
	_, err := repository.pool.Exec(context, query,
		user.ID,
		user.Username,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		user.Website,
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
	query := fmt.Sprintf(`
		UPDATE %s
		SET %s = $2, %s = $3
		WHERE %s = $1 AND %s IS NULL`,
		schema.UserAccount.Table, schema.UserAccount.Password, schema.UserAccount.UpdatedAt,
		schema.UserAccount.ID, schema.UserAccount.DeletedAt,
	)

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
	query := fmt.Sprintf("UPDATE %s SET %s = TRUE, %s = $2 WHERE %s = $1",
		schema.UserAccount.Table, schema.UserAccount.IsVerified, schema.UserAccount.UpdatedAt, schema.UserAccount.ID)
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
	query := fmt.Sprintf(`
		INSERT INTO %s (
			%s, %s, %s, %s, %s, %s, %s, %s
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		schema.UserSession.Table,
		schema.UserSession.ID, schema.UserSession.UserID, schema.UserSession.TokenHash,
		schema.UserSession.UserAgent, schema.UserSession.IPAddress, schema.UserSession.ExpiresAt,
		schema.UserSession.IsRevoked, schema.UserSession.CreatedAt,
	)

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
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1 AND %s = FALSE AND %s > NOW()`,
		schema.UserSession.ID, schema.UserSession.UserID, schema.UserSession.TokenHash,
		schema.UserSession.UserAgent, schema.UserSession.IPAddress, schema.UserSession.ExpiresAt,
		schema.UserSession.IsRevoked, schema.UserSession.CreatedAt,
		schema.UserSession.Table,
		schema.UserSession.TokenHash, schema.UserSession.IsRevoked, schema.UserSession.ExpiresAt,
	)

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
	query := fmt.Sprintf("UPDATE %s SET %s = TRUE WHERE %s = $1",
		schema.UserSession.Table, schema.UserSession.IsRevoked, schema.UserSession.ID)
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
	query := fmt.Sprintf("UPDATE %s SET %s = TRUE WHERE %s = $1 AND %s = FALSE",
		schema.UserSession.Table, schema.UserSession.IsRevoked, schema.UserSession.UserID, schema.UserSession.IsRevoked)
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
	query := fmt.Sprintf("UPDATE %s SET %s = TRUE WHERE %s = $1 AND %s != $2 AND %s = FALSE",
		schema.UserSession.Table, schema.UserSession.IsRevoked, schema.UserSession.UserID,
		schema.UserSession.ID, schema.UserSession.IsRevoked)
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
	query := fmt.Sprintf("DELETE FROM %s WHERE %s <= NOW()", schema.UserSession.Table, schema.UserSession.ExpiresAt)
	_, err := repository.pool.Exec(context, query)
	if err != nil {
		return fmt.Errorf("postgres_session_repo_delete_expired_failed: %w", err)
	}
	return nil
}
