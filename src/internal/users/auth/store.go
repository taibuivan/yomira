// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package auth

import (
	"context"
	"time"
)

// # User Data Access

// UserRepository defines the data access contract for user accounts.
type UserRepository interface {

	/*
		FindByID returns the account with the given ID.

		Parameters:
		  - context: context.Context
		  - id: string

		Returns:
		  - *User: Hydrated entity
		  - error: Database retrieval failures
	*/
	FindByID(context context.Context, id string) (*User, error)

	/*
		FindByEmail returns the account with the given email.

		Parameters:
		  - context: context.Context
		  - email: string

		Returns:
		  - *User: Hydrated entity
		  - error: Database retrieval failures
	*/
	FindByEmail(context context.Context, email string) (*User, error)

	/*
		FindByUsername returns the account with the given username.

		Parameters:
		  - context: context.Context
		  - username: string

		Returns:
		  - *User: Hydrated entity
		  - error: Database retrieval failures
	*/
	FindByUsername(context context.Context, username string) (*User, error)

	/*
		Create persists a brand-new user account to the storage.

		Parameters:
		  - context: context.Context
		  - user: *User

		Returns:
		  - error: Persistence failures
	*/
	Create(context context.Context, user *User) error

	/*
		Update persists changes to mutable profile fields.

		Parameters:
		  - context: context.Context
		  - user: *User

		Returns:
		  - error: Persistence failures
	*/
	Update(context context.Context, user *User) error

	/*
		UpdatePassword replaces only the user's password hash.

		Parameters:
		  - context: context.Context
		  - userID: string
		  - newHash: string

		Returns:
		  - error: Persistence failures
	*/
	UpdatePassword(context context.Context, userID, newHash string) error

	/*
		SoftDelete marks the account as deleted without removing the row.

		Parameters:
		  - context: context.Context
		  - id: string

		Returns:
		  - error: Persistence failures
	*/
	SoftDelete(context context.Context, id string) error

	/*
		MarkVerified updates the user's status to isverified = true.

		Parameters:
		  - context: context.Context
		  - userID: string

		Returns:
		  - error: Persistence failures
	*/
	MarkVerified(context context.Context, userID string) error
}

// # Session Data Access

// SessionRepository defines the data access contract for refresh-token sessions.
type SessionRepository interface {

	/*
		Create persists a new tracking session for an authenticated login.

		Parameters:
		  - context: context.Context
		  - session: *Session

		Returns:
		  - error: Persistence failures
	*/
	Create(context context.Context, session *Session) error

	/*
		FindByTokenHash returns the active session matching the given token hash.

		Parameters:
		  - context: context.Context
		  - tokenHash: string

		Returns:
		  - *Session: Hydrated entity
		  - error: Database retrieval failures
	*/
	FindByTokenHash(context context.Context, tokenHash string) (*Session, error)

	/*
		Revoke marks a specific session as permanently invalidated.

		Parameters:
		  - context: context.Context
		  - sessionID: string

		Returns:
		  - error: Persistence failures
	*/
	Revoke(context context.Context, sessionID string) error

	/*
		RevokeAll revokes every active session belonging to the userID.

		Parameters:
		  - context: context.Context
		  - userID: string

		Returns:
		  - error: Persistence failures
	*/
	RevokeAll(context context.Context, userID string) error

	/*
		RevokeOthers revokes all sessions belonging to the userID except for the current session.

		Parameters:
		  - context: context.Context
		  - userID: string
		  - currentSessionID: string

		Returns:
		  - error: Persistence failures
	*/
	RevokeOthers(context context.Context, userID, currentSessionID string) error

	/*
		DeleteExpired physically removes sessions whose ExpiresAt is in the past.

		Parameters:
		  - context: context.Context

		Returns:
		  - error: Persistence failures
	*/
	DeleteExpired(context context.Context) error
}

// # Volatile Data Access

// ResetTokenRepository defines the contract for storing volatile password reset tokens.
type ResetTokenRepository interface {

	/*
		Set stores a reset token associated with a userID for a limited duration.

		Parameters:
		  - context: context.Context
		  - token: string
		  - userID: string
		  - ttl: time.Duration

		Returns:
		  - error: Persistence failures
	*/
	Set(context context.Context, token string, userID string, ttl time.Duration) error

	/*
		Get retrieves the userID associated with a given reset token.

		Parameters:
		  - context: context.Context
		  - token: string

		Returns:
		  - string: UserID
		  - error: Retrieval failures
	*/
	Get(context context.Context, token string) (string, error)

	/*
		Delete removes a reset token after successful use.

		Parameters:
		  - context: context.Context
		  - token: string

		Returns:
		  - error: Persistence failures
	*/
	Delete(context context.Context, token string) error
}

// VerificationTokenRepository defines the contract for storing volatile email verification tokens.
type VerificationTokenRepository interface {

	/*
		Set stores a verification token associated with a userID.

		Parameters:
		  - context: context.Context
		  - token: string
		  - userID: string
		  - ttl: time.Duration

		Returns:
		  - error: Persistence failures
	*/
	Set(context context.Context, token string, userID string, ttl time.Duration) error

	/*
		Get retrieves the userID associated with a given verification token.

		Parameters:
		  - context: context.Context
		  - token: string

		Returns:
		  - string: UserID
		  - error: Retrieval failures
	*/
	Get(context context.Context, token string) (string, error)

	/*
		Delete removes a verification token after successful use.

		Parameters:
		  - context: context.Context
		  - token: string

		Returns:
		  - error: Persistence failures
	*/
	Delete(context context.Context, token string) error
}
