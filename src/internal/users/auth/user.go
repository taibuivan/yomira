// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package auth implements the user identity and session management layer.

It defines the core domain entities (User, Session) and logic for authentication,
authorization, and account lifecycle.

# Architecture

This layer is the "Truth" of the system. Entities defined here have no external
dependencies and encapsulate all business rules related to user identity.
*/
package auth

import (
	"time"

	"github.com/taibuivan/yomira/internal/platform/sec"
)

// # Domain Entities

// User represents a registered member of the Yomira platform.
type User struct {
	ID           string       `json:"id"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"-"` // Explicitly omitted from JSON for security.
	DisplayName  string       `json:"display_name"`
	AvatarURL    string       `json:"avatar_url,omitempty"`
	Bio          string       `json:"bio,omitempty"`
	Role         sec.UserRole `json:"role"`
	IsVerified   bool         `json:"is_verified"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Session represents an active refresh-token session.
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

// # Field Identifiers

// Global field names for validation and identity mapping in the authentication domain.
const (
	FieldUsername        = "username"
	FieldEmail           = "email"
	FieldPassword        = "password"
	FieldDisplayName     = "display_name"
	FieldLogin           = "login"
	FieldToken           = "token"
	FieldCurrentPassword = "current_password"
	FieldNewPassword     = "new_password"
	FieldAccessToken     = "access_token"
	FieldTokenType       = "token_type"
	FieldExpiresIn       = "expires_in"
	FieldUser            = "user"
	FieldMessage         = "message"
)
