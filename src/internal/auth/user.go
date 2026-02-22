// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package domain defines the core business entities and rules of the Yomira platform.
//
// # Architecture
//
// Entities in this package represent the "Truth" of the system.
// They have no dependencies on outer layers (like databases, APIs, or libraries).
// This makes the core logic highly testable and resilient to technology changes.
package auth

import (
	"time"
)

// UserRole represents the authorization level granted to an account.
//
// # Usage
//
// Used by [middleware.RequireRole] to enforce access control on API endpoints.
type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"     // Unrestricted system access.
	UserRoleModerator UserRole = "moderator" // Can manage community content.
	UserRoleAuthor    UserRole = "author"    // Can upload and manage their own comics.
	UserRoleMember    UserRole = "member"    // Default role for registered users.
)

// level maps a role to a numeric hierarchy level to easily check permissions.
func (r UserRole) level() int {
	switch r {
	case UserRoleAdmin:
		return 40
	case UserRoleModerator:
		return 30
	case UserRoleAuthor:
		return 20
	case UserRoleMember:
		return 10
	default:
		return 0
	}
}

// AtLeast checks if the current role meets or exceeds the required target role.
//
// # Why numeric mapping?
//
// Using numeric levels allows simple >= comparisons instead of nested IF/SWITCH
// statements when deciding if a Moderator has permission to do a Member-level action.
func (r UserRole) AtLeast(target UserRole) bool {
	return r.level() >= target.level()
}

// User represents a registered member of the Yomira platform.

// # Rules
//   - Username is unique and URL-safe.
//   - Email is unique and validated.
//   - PasswordHash is generated via Bcrypt exclusively via AuthService.
//   - IsVerified ensures the user has confirmed their email address.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Explicitly omitted from JSON for security.
	DisplayName  string    `json:"display_name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	Role         UserRole  `json:"role"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an active refresh-token session.
//
// # Security Concept
//
// Access Tokens (JWT) are stateless and cannot be revoked easily before they expire.
// To mitigate this, Yomira uses short-lived JWTs paired with long-lived Sessions
// stored in the database. When the JWT expires, the client uses the Session
// (Refresh Token) to issue a new JWT. Revoking a Session logs the user out globally.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"` // Hashed value of the refresh token. Omitted for security.
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	IsRevoked bool      `json:"is_revoked"`
	CreatedAt time.Time `json:"created_at"`
}
