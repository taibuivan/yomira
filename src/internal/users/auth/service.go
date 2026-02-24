// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package auth implements the core identity and access management (IAM) system.

It handles everything from user registration and secure password hashing to session
lifecycle management via JWT and Refresh tokens (stored in Redis).

Architecture:

  - Service: Orchestrates business logic (Register, Login, MFA).
  - Repository: Abstracted interfaces for Postgres (Users) and Redis (Sessions).
  - Security: Leverages Argon2/Bcrypt and RSA-signed JWTs.

The package ensures that identity data remains consistent and secure throughout
the platform’s lifecycle.
*/
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/pkg/uuid"
)

// # Contracts & Types

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
	//   - A signed JWT string, or an err if signing fails.
	GenerateAccessToken(userID, username, role string, timeToLive time.Duration) (string, error)
}

// Service implements user authentication use cases.
//
// # Review Process
//
// This service is critical for security. Any changes to hashing, registration,
// or login logic must be reviewed by the security team.
type Service struct {
	userRepository              UserRepository
	sessionRepository           SessionRepository
	resetTokenRepository        ResetTokenRepository
	verificationTokenRepository VerificationTokenRepository
	tokenProvider               TokenProvider
}

// NewService constructs a new [AuthService] with necessary dependencies.
func NewService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	resetRepo ResetTokenRepository,
	verifyRepo VerificationTokenRepository,
	tokenProv TokenProvider,
) *Service {
	return &Service{
		userRepository:              userRepo,
		sessionRepository:           sessionRepo,
		resetTokenRepository:        resetRepo,
		verificationTokenRepository: verifyRepo,
		tokenProvider:               tokenProv,
	}
}

// # Registration Flow

// RegisterInput holds the data required to enroll a new member.
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

/*
Register validates, hashes, and persists a brand new user account.

Description: Deep-enrollment of a new member, handling password hashing and
initial verification token state.

Parameters:
  - context: context.Context
  - input: RegisterInput

Returns:
  - *User: Created entity
  - err: Conflict (if identity exists) or storage errors
*/
func (service *Service) Register(context context.Context, input RegisterInput) (*User, error) {

	// Verify email uniqueness. Return a client-safe Conflict err.
	_, err := service.userRepository.FindByEmail(context, input.Email)
	if err == nil {
		return nil, apperr.Conflict("Email is already registered")
	}

	// Verify username uniqueness. Return a client-safe Conflict err.
	_, err = service.userRepository.FindByUsername(context, input.Username)
	if err == nil {
		return nil, apperr.Conflict("Username is already taken")
	}

	// Prevent storing plain-text passwords. Default cost is used for balance
	// between security and CPU utilization during registration spikes.
	hashedPassword, err := sec.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("auth_service_hash_failed: %w", err)
	}

	// Construct the new User entity. Time-sortable ID to prevent PG index fragmentation.
	user := &User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: hashedPassword,
		DisplayName:  input.DisplayName,
		Role:         sec.RoleMember,
		IsVerified:   false,
	}

	// Persist the user to the database
	if err := service.userRepository.Create(context, user); err != nil {
		return nil, fmt.Errorf("auth_service_register_failed: %w", err)
	}

	// Generate and store a verification token in Redis as an async-ready side effect
	token, err := sec.GenerateSecureToken(VerificationTokenLength)
	if err == nil {
		_ = service.verificationTokenRepository.Set(context, token, user.ID, VerificationTokenTTL)
		// TODO: Trigger email service with the verification link
	}

	return user, nil
}

// # Authentication Flow

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

/*
Login validates user credentials and issues security tokens.

Description: Verifies identity, performs constant-time password comparison,
and initializes a new session with rotated security tokens.

Parameters:
  - context: context.Context
  - input: LoginInput

Returns:
  - *LoginSession: Transport-ready session identifiers
  - err: Unauthorized or internal failures
*/
func (service *Service) Login(context context.Context, input LoginInput) (*LoginSession, error) {
	var user *User
	var err error
	// Flexible login: look up by Email or Username
	user, err = service.userRepository.FindByEmail(context, input.Login)
	if err != nil {
		user, err = service.userRepository.FindByUsername(context, input.Login)
	}

	// If (err != nil) the user does not exist. Generic message to prevent enumeration.
	if err != nil {
		return nil, apperr.Unauthorized("Invalid login credentials")
	}

	// Verify password hash usando constant-time comparison in bcrypt to prevent timing attacks
	if !sec.CheckPasswordHash(input.Password, user.PasswordHash) {
		return nil, apperr.Unauthorized("Invalid login credentials")
	}

	// Generate short-lived Access Token
	accessToken, err := service.tokenProvider.GenerateAccessToken(user.ID, user.Username, string(user.Role), AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("auth_service_token_generation_failed: %w", err)
	}

	// Generate long-lived Refresh Token
	refreshToken, err := sec.GenerateSecureToken(RefreshTokenLength)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_token_failed: %w", err)
	}

	// Create and persist the tracking session
	expiresAt := time.Now().Add(RefreshTokenTTL)
	session := &Session{
		ID:        uuid.New(),
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

/*
Logout permanently revokes the user's active session.

Description: Ensures that a tracked refresh token can never be used again.

Parameters:
  - context: context.Context
  - refreshToken: string

Returns:
  - err: Revocation failures
*/
func (service *Service) Logout(context context.Context, refreshToken string) error {

	// Hash the refresh token
	tokenHash := sec.HashToken(refreshToken)

	// Find the session by token hash
	session, err := service.sessionRepository.FindByTokenHash(context, tokenHash)

	// If (err != nil) session is already gone or invalid, we consider logout successful (idempotent operation).
	if err != nil {
		return nil
	}

	// If (err == nil) Revoke the session
	if err := service.sessionRepository.Revoke(context, session.ID); err != nil {
		return fmt.Errorf("auth_service_logout_failed: %w", err)
	}

	return nil
}

// # Session Management

/*
RefreshSession implements the Refresh Token Rotation mechanism.

Description: Verifies the existing refresh token, revokes it to prevent reuse
(replay attack mitigation), and issues a fresh pair of rotated tokens.

Parameters:
  - context: context.Context
  - refreshToken: string
  - userAgent: string
  - ipAddress: string

Returns:
  - *LoginSession: New session credentials
  - err: Unauthorized or storage failures
*/
func (service *Service) RefreshSession(context context.Context, refreshToken, userAgent, ipAddress string) (*LoginSession, error) {

	// Hash the incoming refresh token to look it up
	tokenHash := sec.HashToken(refreshToken)
	session, err := service.sessionRepository.FindByTokenHash(context, tokenHash)

	// If (err != nil) the token is either expired, already revoked, or completely invalid.
	if err != nil {
		return nil, apperr.Unauthorized("Invalid or expired refresh token")
	}

	// Rotation: Revoke the old session to prevent replay attacks
	if err := service.sessionRepository.Revoke(context, session.ID); err != nil {
		return nil, fmt.Errorf("auth_service_refresh_revoke_failed: %w", err)
	}

	// Fetch the user associated with this session
	user, err := service.userRepository.FindByID(context, session.UserID)
	if err != nil {
		return nil, apperr.Unauthorized("User not found or suspended")
	}

	// Generate a fresh Access Token
	accessToken, err := service.tokenProvider.GenerateAccessToken(user.ID, user.Username, string(user.Role), AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_access_token_failed: %w", err)
	}

	// Generate a fresh Refresh Token for the rotation
	newRefreshToken, err := sec.GenerateSecureToken(RefreshTokenLength)
	if err != nil {
		return nil, fmt.Errorf("auth_service_refresh_secure_token_failed: %w", err)
	}

	// Persist the new session
	expiresAt := time.Now().Add(RefreshTokenTTL)
	newSession := &Session{
		ID:        uuid.New(),
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

// # Password Recovery

/*
RequestPasswordReset initiates the forgot-password flow.

Description: Generates a secure token and saves it to Redis.

Parameters:
  - context: context.Context
  - email: string

Returns:
  - string: Discovery token
  - err: Generation errors
*/
func (service *Service) RequestPasswordReset(context context.Context, email string) (string, error) {
	// Look up user.
	// NOTE: We don't return NOT_FOUND if the email doesn't exist to prevent user enumeration.
	user, err := service.userRepository.FindByEmail(context, email)
	if err != nil {
		return "", nil
	}

	// Generate reset token
	token, err := sec.GenerateSecureToken(ResetTokenLength)
	if err != nil {
		return "", fmt.Errorf("auth_service_generate_reset_token_failed: %w", err)
	}

	// Save to Redis
	if err := service.resetTokenRepository.Set(context, token, user.ID, ResetTokenTTL); err != nil {
		return "", fmt.Errorf("auth_service_save_reset_token_failed: %w", err)
	}

	return token, nil
}

/*
ResetPassword completes the forgot-password flow.

Description: Verifies the token, hashes the new password, updates the DB,
and revokes all active sessions for security cleanup.

Parameters:
  - context: context.Context
  - token: string
  - newPassword: string

Returns:
  - err: Validation or update failures
*/
func (service *Service) ResetPassword(context context.Context, token, newPassword string) error {

	// Retrieve the userID associated with the reset token from Redis
	userID, err := service.resetTokenRepository.Get(context, token)
	if err != nil {
		return err
	}

	// Hash the new password securely
	hashedPassword, err := sec.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth_service_reset_password_hash_failed: %w", err)
	}

	// Update the user's password in persistent storage
	if err := service.userRepository.UpdatePassword(context, userID, hashedPassword); err != nil {
		return fmt.Errorf("auth_service_reset_password_update_failed: %w", err)
	}

	// Security Cleanup: Revoke EVERY active session for this user
	_ = service.sessionRepository.RevokeAll(context, userID)

	// Delete the used token from Redis
	_ = service.resetTokenRepository.Delete(context, token)

	return nil
}

/*
ChangePassword allows an authenticated user to update their credentials.

Description: Verifies the current password and then rotates all OTHER refresh sessions
to ensure high security across devices.

Parameters:
  - context: context.Context
  - userID: string
  - currentPassword: string
  - newPassword: string
  - currentRefreshToken: string

Returns:
  - err: Unauthorized or storage failures
*/
func (service *Service) ChangePassword(context context.Context, userID, currentPassword, newPassword, currentRefreshToken string) error {

	// Fetch user by ID
	user, err := service.userRepository.FindByID(context, userID)
	if err != nil {
		return err
	}

	// Verify the current password before allowing change
	if !sec.CheckPasswordHash(currentPassword, user.PasswordHash) {
		return apperr.Unauthorized("Current password is incorrect")
	}

	// Hash the brand new password
	hashedPassword, err := sec.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth_service_change_password_hash_failed: %w", err)
	}

	// Update the database with the new hash
	if err := service.userRepository.UpdatePassword(context, userID, hashedPassword); err != nil {
		return fmt.Errorf("auth_service_change_password_update_failed: %w", err)
	}

	// Security Side Effect: Revoke all other sessions to force re-login on other devices
	tokenHash := sec.HashToken(currentRefreshToken)
	session, err := service.sessionRepository.FindByTokenHash(context, tokenHash)
	if err == nil {
		_ = service.sessionRepository.RevokeOthers(context, userID, session.ID)
	}

	return nil
}

/*
VerifyEmail confirms a user's email address using a secure token.

Parameters:
  - context: context.Context
  - token: string

Returns:
  - err: Database or resolution errors
*/
func (service *Service) VerifyEmail(context context.Context, token string) error {

	// Retrieve the user ID associated with the verification token from Redis
	userID, err := service.verificationTokenRepository.Get(context, token)
	if err != nil {
		return err
	}

	// Update the user's status to verified in persistent storage
	if err := service.userRepository.MarkVerified(context, userID); err != nil {
		return fmt.Errorf("auth_service_verify_email_failed: %w", err)
	}

	// Cleanup the used verification token from Redis
	_ = service.verificationTokenRepository.Delete(context, token)

	return nil
}
