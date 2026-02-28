// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package account handles user profile management, preferences, and security settings.

It provides functionalities for users to view and update their private identity data,
configure their reading experience, and manage their active device sessions.

# Architecture

  - Entities: Preferences, SessionInfo (DTO).
  - Domain: This package depends on the auth package for the User entity.
  - Security: Provides session transparency and revocation mechanisms.
*/
package account

import (
	"context"
	"time"

	"github.com/taibuivan/yomira/internal/users/auth"
)

// # Domain Entities

// Preferences represents the customizable reader and UI settings for a user.
type Preferences struct {
	UserID        string    `json:"user_id"`
	ReadingMode   string    `json:"reading_mode"` // 'ltr', 'rtl', 'vertical', 'webtoon'
	PageFit       string    `json:"page_fit"`     // 'width', 'height', 'original', 'stretch'
	DoublePageOn  bool      `json:"double_page_on"`
	ShowPageBar   bool      `json:"show_page_bar"`
	PreloadPages  int       `json:"preload_pages"`  // Performance setting: 1-10 pages
	DataSaver     bool      `json:"data_saver"`     // If true, request optimized image assets
	HideNSFW      bool      `json:"hide_nsfw"`      // Global content filter
	HideLanguages []string  `json:"hide_languages"` // List of BCP-47 language codes to filter out
	UpdatedAt     time.Time `json:"updated_at"`
}

// SessionInfo provides a safety-mapped view of an active user session.
// It omits sensitive token hashes for transport.
type SessionInfo struct {
	ID         string    `json:"id"`
	DeviceName string    `json:"device_name"` // e.g. "Chrome on Windows"
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsCurrent  bool      `json:"is_current"` // True if this session belongs to the current request
}

// # Repository Contracts

// AccountRepository defines the persistence contract for user accounts.
type AccountRepository interface {
	/*
		FindByID retrieves a user record by their unique ID.

		Parameters:
		  - context: context.Context
		  - id: string (UUID)

		Returns:
		  - *User: Loaded account entity
		  - error: apperr.NotFound or storage failures
	*/
	FindByID(context context.Context, id string) (*auth.User, error)

	/*
		Update modifies the mutable profile fields of an existing user.

		Parameters:
		  - context: context.Context
		  - user: *User (Hydrated entity with changes)

		Returns:
		  - error: Storage or constraint failures
	*/
	Update(context context.Context, user *auth.User) error

	/*
		SoftDelete flags an account as logically deleted.

		Parameters:
		  - context: context.Context
		  - id: string

		Returns:
		  - error: Execution failures
	*/
	SoftDelete(context context.Context, id string) error
}

// PreferencesRepository defines the persistence contract for reader settings.
type PreferencesRepository interface {
	/*
		FindByUserID retrieves reader preferences for a specific user.

		Parameters:
		  - context: context.Context
		  - userID: string

		Returns:
		  - *Preferences: Hydrated settings
		  - error: apperr.NotFound if not present
	*/
	FindByUserID(context context.Context, userID string) (*Preferences, error)

	/*
		Upsert saves or updates preferences for a user using an idempotent strategy.

		Parameters:
		  - context: context.Context
		  - prefs: *Preferences

		Returns:
		  - error: Storage failure errors
	*/
	Upsert(context context.Context, prefs *Preferences) error
}

// SessionRepository defines the visibility and revocation contract for user sessions.
type SessionRepository interface {
	/*
		FindActiveByUserID lists all valid, non-expired sessions for a user.

		Parameters:
		  - context: context.Context
		  - userID: string

		Returns:
		  - []SessionInfo: List of active devices
		  - error: Retrieval errors
	*/
	FindActiveByUserID(context context.Context, userID string) ([]SessionInfo, error)

	/*
		Revoke marks a specific session as revoked.

		Parameters:
		  - context: context.Context
		  - userID: string (Security constraint: owner validation)
		  - sessionID: string

		Returns:
		  - error: Revocation failures
	*/
	Revoke(context context.Context, userID, sessionID string) error

	/*
		RevokeOthers revokes all active sessions except for a target session.

		Parameters:
		  - context: context.Context
		  - userID: string
		  - currentSessionID: string (The whitelist target)

		Returns:
		  - error: Revocation failures
	*/
	RevokeOthers(context context.Context, userID, currentSessionID string) error

	/*
		RevokeAll terminates every session for a user.

		Parameters:
		  - context: context.Context
		  - userID: string

		Returns:
		  - error: Revocation failures
	*/
	RevokeAll(context context.Context, userID string) error
}
