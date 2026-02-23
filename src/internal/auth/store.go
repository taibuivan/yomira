// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package auth

import (
	"context"
	"time"
)

// UserRepository defines the data access contract for user accounts.
//
// # Review Process
//
// This interface is placed in a separate file from user.go so entity changes
// and storage-contract changes can be reviewed independently by the team.
//
// # Implementations
//
// The canonical implementation for Yomira is PostgreSQL (package: `postgres`).
type UserRepository interface {
	// FindByID returns the account with the given ID.
	//
	// Returns [apperr.NotFound] if the account does not exist.
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail returns the account with the given email.
	//
	// Returns [apperr.NotFound] if no user is registered with this email.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByUsername returns the account with the given username.
	//
	// Returns [apperr.NotFound] if the username is available.
	FindByUsername(ctx context.Context, username string) (*User, error)

	// Create persists a brand-new user account to the storage.
	//
	// Returns a wrapped error if a unique constraint (email/username) fails.
	Create(ctx context.Context, user *User) error

	// Update persists changes to mutable profile fields (DisplayName, Bio, etc).
	// Passwords must be updated via [UpdatePassword].
	Update(ctx context.Context, user *User) error

	// UpdatePassword replaces only the user's password hash.
	// This is separate from [Update] to prevent accidental overwrites
	// during unrelated profile updates.
	UpdatePassword(ctx context.Context, userID, newHash string) error

	// SoftDelete marks the account as deleted without removing the row.
	// This preserves relational integrity (e.g., comments left by the user).
	SoftDelete(ctx context.Context, id string) error
}

// SessionRepository defines the data access contract for refresh-token sessions.
//
// # Domain Ownership
//
// This is kept alongside [UserRepository] because sessions are owned entirely
// by the users' domain, despite serving authentication security.
type SessionRepository interface {
	// Create persists a new tracking session for an authenticated login.
	Create(ctx context.Context, session *Session) error

	// FindByTokenHash returns the active session matching the given token hash.
	//
	// Returns [apperr.NotFound] if the session is invalid, expired, or revoked.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)

	// Revoke marks a specific session as permanently invalidated.
	// Usually triggered during explicit user logout from a specific device.
	Revoke(ctx context.Context, sessionID string) error

	// RevokeAll revokes every active session belonging to the userID.
	// Crucial for security event responses (e.g., password change or account compromise).
	RevokeAll(ctx context.Context, userID string) error

	// DeleteExpired physically removes sessions whose ExpiresAt is in the past.
	// Intended to be called by a periodic background cleanup worker to reclaim storage.
	DeleteExpired(ctx context.Context) error
}

// ResetTokenRepository defines the contract for storing volatile password reset tokens.
type ResetTokenRepository interface {
	// Set stores a reset token associated with a userID for a limited duration.
	Set(ctx context.Context, token string, userID string, ttl time.Duration) error

	// Get retrieves the userID associated with a given reset token.
	Get(ctx context.Context, token string) (string, error)

	// Delete removes a reset token after successful use.
	Delete(ctx context.Context, token string) error
}
