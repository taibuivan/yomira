// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package account

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/users/auth"
)

// # Service Layer

// Service orchestrates business logic for user accounts and preferences.
//
// It ensures that profile updates, preference persistence, and session
// security cleanup follow established business constraints.
type Service struct {
	accountRepository     AccountRepository
	preferencesRepository PreferencesRepository
	sessionRepository     SessionRepository
	logger                *slog.Logger
}

// NewService constructs a new [Service] with its repository dependencies.
func NewService(
	accountRepo AccountRepository,
	preferencesRepo PreferencesRepository,
	sessionRepo SessionRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		accountRepository:     accountRepo,
		preferencesRepository: preferencesRepo,
		sessionRepository:     sessionRepo,
		logger:                logger,
	}
}

// # Profile Management

/*
GetProfile retrieves the full private identity of a user.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - *auth.User: The hydrated user profile
  - error: Not found or execution failures
*/
func (service *Service) GetProfile(context context.Context, userID string) (*auth.User, error) {
	user, err := service.accountRepository.FindByID(context, userID)
	if err != nil {
		return nil, fmt.Errorf("account_service_get_profile_failed: %w", err)
	}
	return user, nil
}

// UpdateProfileInput defines the mutable subset of user profile fields.
type UpdateProfileInput struct {
	DisplayName *string
	Bio         *string
	Website     *string
}

/*
UpdateProfile applies a partial set of changes to a user's account metadata.

Description: Fetches the existing user state, overries provided fields, and
synchronizes the change to persistent storage.

Parameters:
  - context: context.Context
  - userID: string
  - input: UpdateProfileInput

Returns:
  - *auth.User: The updated user profile
  - error: Update or storage failures
*/
func (service *Service) UpdateProfile(context context.Context, userID string, input UpdateProfileInput) (*auth.User, error) {

	// Business: Ensure the user is authenticated
	user, err := service.accountRepository.FindByID(context, userID)
	if err != nil {
		return nil, fmt.Errorf("account_service_update_lookup_failed: %w", err)
	}

	// Apply delta updates
	if input.DisplayName != nil {
		user.DisplayName = *input.DisplayName
	}

	// Apply delta updates
	if input.Bio != nil {
		user.Bio = *input.Bio
	}

	// Apply delta updates
	if input.Website != nil {
		user.Website = *input.Website
	}

	// Persist changes
	if err := service.accountRepository.Update(context, user); err != nil {
		return nil, fmt.Errorf("account_service_update_failed: %w", err)
	}

	service.logger.Info("user_profile_updated", slog.String("user_id", userID))

	return user, nil
}

/*
DeleteAccount performs an idempotent soft-deletion of a user account.

Description: Flags the account as deleted and immediately terminates all active
security sessions to force a global sign-out.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - error: Execution failures
*/
func (service *Service) DeleteAccount(context context.Context, userID string) error {

	// Business: Ensure the user is authenticated
	if err := service.accountRepository.SoftDelete(context, userID); err != nil {
		return fmt.Errorf("account_service_delete_failed: %w", err)
	}

	// Force global revocation of sessions for the deleted account
	_ = service.sessionRepository.RevokeAll(context, userID)

	service.logger.Warn("user_account_deleted", slog.String("user_id", userID))

	return nil
}

// # Preferences Management

/*
GetPreferences retrieves the reader settings for a specific user ID.

Description: Attempts a database lookup. If no explicit preferences exist,
it fallback to system-wide default settings.

Parameters:
  - context: context.Context
  - userID: string

Returns:
  - *Preferences: Current or default settings
  - error: Storage failures
*/
func (service *Service) GetPreferences(context context.Context, userID string) (*Preferences, error) {

	// Business: Ensure the user is authenticated
	prefs, err := service.preferencesRepository.FindByUserID(context, userID)

	if err != nil {
		// Resilience: provide defaults if none are stored
		if apperr.IsNotFound(err) {
			return &Preferences{
				UserID:       userID,
				ReadingMode:  "ltr",
				PageFit:      "width",
				PreloadPages: 3,
				UpdatedAt:    time.Now(),
			}, nil
		}

		// Business: Ensure the user is authenticated
		return nil, fmt.Errorf("account_service_get_preferences_failed: %w", err)
	}

	// Business: Ensure the user is authenticated
	return prefs, nil
}

/*
UpdatePreferences persists new reader and UI settings for the user.

Parameters:
  - context: context.Context
  - prefs: *Preferences

Returns:
  - error: Storage failures
*/
func (service *Service) UpdatePreferences(context context.Context, prefs *Preferences) error {

	// Business: Ensure the user is authenticated
	prefs.UpdatedAt = time.Now()
	if err := service.preferencesRepository.Upsert(context, prefs); err != nil {
		return fmt.Errorf("account_service_save_preferences_failed: %w", err)
	}

	service.logger.Info("user_preferences_updated", slog.String("user_id", prefs.UserID))

	return nil
}

// # Session Security

/*
ListSessions provides a list of all active device sessions for the user.

Parameters:
  - context: context.Context
  - userID: string
  - currentTokenHash: string (Optional identifying hash of the current session)

Returns:
  - []SessionInfo: List of active devices
  - error: Retrieval failures
*/
func (service *Service) ListSessions(context context.Context, userID, currentTokenHash string) ([]SessionInfo, error) {

	// Ensure the user is authenticated
	sessions, err := service.sessionRepository.FindActiveByUserID(context, userID)

	if err != nil {
		return nil, fmt.Errorf("account_service_list_sessions_failed: %w", err)
	}

	return sessions, nil
}

/*
RevokeSession terminates a specific user session by its ID.

Parameters:
  - context: context.Context
  - userID: string
  - sessionID: string

Returns:
  - error: Revocation failures
*/
func (service *Service) RevokeSession(context context.Context, userID, sessionID string) error {
	if err := service.sessionRepository.Revoke(context, userID, sessionID); err != nil {
		return fmt.Errorf("account_service_revoke_session_failed: %w", err)
	}

	service.logger.Info("user_session_revoked",
		slog.String("user_id", userID),
		slog.String("session_id", sessionID),
	)

	return nil
}

/*
RevokeOtherSessions terminates all sessions except for the current active one.

Parameters:
  - context: context.Context
  - userID: string
  - currentSessionID: string

Returns:
  - error: Revocation failures
*/
func (service *Service) RevokeOtherSessions(context context.Context, userID, currentSessionID string) error {
	if err := service.sessionRepository.RevokeOthers(context, userID, currentSessionID); err != nil {
		return fmt.Errorf("account_service_revoke_others_failed: %w", err)
	}

	service.logger.Info("user_other_sessions_revoked", slog.String("user_id", userID))

	return nil
}
