// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package application implements the core business logic (Use Cases) for the Yomira platform.
//
// # Architecture
//
// Services in this package orchestrate domain entities and interact with
// repositories through interfaces. They are technology-agnostic and do not
// know about HTTP or SQL.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/pkg/uuidv7"
)

// TokenProvider defines the contract for generating security tokens.
type TokenProvider interface {
	// GenerateAccessToken creates a signed JWT string for the given user.
	//
	// # Parameters
	//   - userID: The ID of the account.
	//   - username: The username of the account.
	//   - role: The role of the account.
	//   - timeToLive: The duration before the token expires.
	//
	// # Returns
	//   - A signed JWT string, or an error if signing fails.
	GenerateAccessToken(userID, username, role string, timeToLive time.Duration) (string, error)
}

// Service implements user authentication use cases.
//
// # Review Process
//
// This service is critical for security. Any changes to hashing, registration,
// or login logic must be reviewed by the security team.
type Service struct {
	userRepository    UserRepository
	sessionRepository SessionRepository
	tokenProvider     TokenProvider
}

// NewService constructs a new [AuthService] with necessary dependencies.
func NewService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	tokenProv TokenProvider,
) *Service {
	return &Service{
		userRepository:    userRepo,
		sessionRepository: sessionRepo,
		tokenProvider:     tokenProv,
	}
}

// RegisterInput holds the data required to enroll a new member.
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// Register validates, hashes, and persists a brand new user account.
//
// # Parameters
//   - context: Context for the database operation.
//   - input: The user-provided registration details.
//
// # Returns
//   - A pointer to the newly created [*User].
//   - Returns [apperr.Conflict] if email or username already exists.
//
// # Business Rules
//   - Emails must be unique.
//   - Usernames must be unique.
//   - Default role is always 'member'.
func (service *Service) Register(context context.Context, input RegisterInput) (*User, error) {
	// ── 1. Uniqueness Checks ──────────────────────────────────────────────

	// Verify email uniqueness. Return a client-safe Conflict error.
	_, err := service.userRepository.FindByEmail(context, input.Email)
	if err == nil {
		return nil, apperr.Conflict("Email is already registered")
	}

	// Verify username uniqueness. Return a client-safe Conflict error.
	_, err = service.userRepository.FindByUsername(context, input.Username)
	if err == nil {
		return nil, apperr.Conflict("Username is already taken")
	}

	// ── 2. Security ───────────────────────────────────────────────────────

	// Prevent storing plain-text passwords. Default cost is used for balance
	// between security and CPU utilization during registration spikes.
	hashedPassword, err := sec.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("auth_service_hash_failed: %w", err)
	}

	// ── 3. Entity Construction ────────────────────────────────────────────

	user := &User{
		ID:           uuidv7.New(), // Time-sortable ID to prevent PG index fragmentation.
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hashedPassword,
		DisplayName:  input.DisplayName,
		Role:         UserRoleMember, // Rule: Default role is always Member
		IsVerified:   false,
	}

	// ── 4. Persistence ────────────────────────────────────────────────────

	if err := service.userRepository.Create(context, user); err != nil {
		return nil, fmt.Errorf("auth_service_register_failed: %w", err)
	}

	return user, nil
}

// LoginInput defines credentials for an authentication attempt.
type LoginInput struct {
	Login     string // Can be Username or Email
	Password  string
	UserAgent string
	IPAddress string
}

// LoginSession represents a successfully established user session.
type LoginSession struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  *User
}

// Login validates user credentials and issues security tokens.
//
// # Parameters
//   - context: Context for the database operation.
//   - input: Contains Login (Email/Username) and plain-text Password.
//
// # Returns
//   - A pointer to [LoginSession] containing the AccessToken.
//   - Returns [apperr.Unauthorized] if credentials do not match.
//
// # Flow
//  1. Lookup user by login (email or username).
//  2. Verify password hash using Bcrypt.
//  3. Generate long-lived JWT access token.
func (service *Service) Login(context context.Context, input LoginInput) (*LoginSession, error) {
	var user *User
	var err error

	// ── 1. Fetch User Profile ─────────────────────────────────────────────

	// We support flexible login, allowing the user to use either Email or Username.
	user, err = service.userRepository.FindByEmail(context, input.Login)
	if err != nil {
		user, err = service.userRepository.FindByUsername(context, input.Login)
	}

	// Return generic unauthorized error to prevent username enumeration attacks.
	if err != nil {
		return nil, apperr.Unauthorized("Invalid login credentials")
	}

	// ── 2. Security Verification ──────────────────────────────────────────

	// Prevent timing attacks by always using constant-time comparison in bcrypt.
	if !sec.CheckPasswordHash(input.Password, user.PasswordHash) {
		return nil, apperr.Unauthorized("Invalid login credentials")
	}

	// ── 3. Token Issuance ─────────────────────────────────────────────────

	// Tokens are valid for 15 minutes to reduce impact window if leaked.
	accessToken, err := service.tokenProvider.GenerateAccessToken(user.ID, user.Username, string(user.Role), 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("auth_service_token_generation_failed: %w", err)
	}

	// ── 4. Refresh Token Issuance ─────────────────────────────────────────

	refreshToken, err := sec.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_token_failed: %w", err)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Valid for 30 days
	session := &Session{
		ID:        uuidv7.New(),
		UserID:    user.ID,
		TokenHash: sec.HashToken(refreshToken),
		UserAgent: input.UserAgent,
		IPAddress: input.IPAddress,
		ExpiresAt: expiresAt,
		IsRevoked: false,
	}

	if err := service.sessionRepository.Create(context, session); err != nil {
		return nil, fmt.Errorf("auth_service_session_creation_failed: %w", err)
	}

	return &LoginSession{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: expiresAt,
		User:                  user,
	}, nil
}

// Logout permanently revokes the user's active session.
// This ensures that the tracked refresh token can never be used again.
func (service *Service) Logout(context context.Context, refreshToken string) error {
	tokenHash := sec.HashToken(refreshToken)
	session, err := service.sessionRepository.FindByTokenHash(context, tokenHash)
	if err != nil {
		// If session is already gone or invalid, we consider logout successful (idempotent operation).
		return nil
	}

	if err := service.sessionRepository.Revoke(context, session.ID); err != nil {
		return fmt.Errorf("auth_service_logout_failed: %w", err)
	}

	return nil
}

// RefreshSession implements the Refresh Token Rotation mechanism.
// It verifies the existing refresh token, revokes it to prevent reuse (preventing replay attacks),
// and issues a fresh pair of Access and Refresh tokens.
func (service *Service) RefreshSession(context context.Context, refreshToken, userAgent, ipAddress string) (*LoginSession, error) {
	// ── 1. Find Existing Session ──────────────────────────────────────────

	tokenHash := sec.HashToken(refreshToken)
	session, err := service.sessionRepository.FindByTokenHash(context, tokenHash)
	if err != nil {
		// The token is either expired, already revoked, or completely invalid.
		return nil, apperr.Unauthorized("Invalid or expired refresh token")
	}

	// ── 2. Rotation (Revoke Old Session) ──────────────────────────────────

	if err := service.sessionRepository.Revoke(context, session.ID); err != nil {
		return nil, fmt.Errorf("auth_service_refresh_revoke_failed: %w", err)
	}

	// ── 3. Find User ──────────────────────────────────────────────────────

	user, err := service.userRepository.FindByID(context, session.UserID)
	if err != nil {
		return nil, apperr.Unauthorized("User not found or suspended")
	}

	// ── 4. Issue New Tokens ───────────────────────────────────────────────

	accessToken, err := service.tokenProvider.GenerateAccessToken(user.ID, user.Username, string(user.Role), 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_access_token_failed: %w", err)
	}

	newRefreshToken, err := sec.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_secure_token_failed: %w", err)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	newSession := &Session{
		ID:        uuidv7.New(),
		UserID:    user.ID,
		TokenHash: sec.HashToken(newRefreshToken),
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: expiresAt,
		IsRevoked: false,
	}

	if err := service.sessionRepository.Create(context, newSession); err != nil {
		return nil, fmt.Errorf("auth_service_refresh_session_creation_failed: %w", err)
	}

	return &LoginSession{
		AccessToken:           accessToken,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: expiresAt,
		User:                  user,
	}, nil
}
