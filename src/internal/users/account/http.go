// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package account provides the HTTP delivery layer for profile and session management.

It implements the RESTful interface for users to interact with their account data,
preferences, and active sessions.

# Security

All endpoints in this package require an active authentication session provided
by the RequireAuth middleware.
*/
package account

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/apperr"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/validate"
)

// Handler implements the HTTP layer for user account management.
type Handler struct {
	accountService *Service
}

// NewHandler constructs a new account [Handler].
func NewHandler(service *Service) *Handler {
	return &Handler{accountService: service}
}

// Routes returns a [chi.Router] configured with the account domain's endpoints.
func (handler *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	// Account Management
	router.Get("/me", handler.getMe)
	router.Patch("/me", handler.updateMe)
	router.Delete("/me", handler.deleteMe)

	// User Preferences
	router.Get("/me/preferences", handler.getPreferences)
	router.Put("/me/preferences", handler.updatePreferences)

	// Session Security
	router.Get("/me/sessions", handler.listSessions)
	router.Delete("/me/sessions", handler.revokeOtherSessions)
	router.Delete("/me/sessions/{id}", handler.revokeSession)

	// Public Profile discovery
	router.Get("/users/{id}", handler.getUserProfile)

	return router
}

// # User Profile Endpoints

/*
GET /api/v1/me.

Description: Retrieves the full private profile of the authenticated user.

Response:
  - 200: User: Fully hydrated user profile
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) getMe(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	user, err := handler.accountService.GetProfile(request.Context(), userID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, user)
}

// updateMeRequest defines the expected JSON payload for profile updates.
type updateMeRequest struct {
	DisplayName *string `json:"display_name"`
	Bio         *string `json:"bio"`
	Website     *string `json:"website"`
}

/*
PATCH /api/v1/me.

Description: Applies partial updates to the authenticated user's profile.

Request:
  - body: updateMeRequest (Partial JSON)

Response:
  - 200: User: The updated profile
  - 400: ErrInvalidJSON/Validation: Invalid input data
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) updateMe(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input updateMeRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	if input.DisplayName != nil {
		v.MinLen("display_name", *input.DisplayName, 2).MaxLen("display_name", *input.DisplayName, 50)
	}
	if input.Bio != nil {
		v.MaxLen("bio", *input.Bio, 500)
	}
	if input.Website != nil && *input.Website != "" {
		v.URL("website", *input.Website)
	}

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	user, err := handler.accountService.UpdateProfile(request.Context(), userID, UpdateProfileInput{
		DisplayName: input.DisplayName,
		Bio:         input.Bio,
		Website:     input.Website,
	})
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, user)
}

/*
DELETE /api/v1/me.

Description: Performs a soft-deletion of the authenticated user's account.

Response:
  - 204: No Content: Account deleted successfully
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) deleteMe(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.accountService.DeleteAccount(request.Context(), userID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}

/*
GET /api/v1/users/{id}.

Description: Retrieves public profile information for a specific user.

Request:
  - id: string (UUID)

Response:
  - 200: User: Public profile data
  - 404: ErrNotFound: User not found or account private
*/
func (handler *Handler) getUserProfile(writer http.ResponseWriter, request *http.Request) {

	// Get user ID
	userID := chi.URLParam(request, "id")

	// If the user ID is empty, return an error
	if userID == "" {
		respond.Error(writer, request, apperr.NotFound("User not found"))
		return
	}

	// Get user profile
	user, err := handler.accountService.GetProfile(request.Context(), userID)

	// If the user is not found, return an error
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// Security: Consider filtering fields for public consumption
	respond.OK(writer, user)
}

// # User Preferences Endpoints

/*
GET /api/v1/me/preferences.

Description: Retrieves the current user's reader and UI settings.

Response:
  - 200: Preferences: Hydraated setting entity
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) getPreferences(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	prefs, err := handler.accountService.GetPreferences(request.Context(), userID)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, prefs)
}

/*
PUT /api/v1/me/preferences.

Description: Overwrites the authenticated user's reader settings.

Request:
  - body: Preferences: Full settings object

Response:
  - 200: Preferences: The persisted settings
  - 400: ErrInvalidJSON: Bad input
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) updatePreferences(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	var input Preferences
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	v := &validate.Validator{}
	v.OneOf("reading_mode", input.ReadingMode, "ltr", "rtl", "vertical", "webtoon").
		OneOf("page_fit", input.PageFit, "width", "height", "original", "stretch").
		Range("preload_pages", input.PreloadPages, 1, 10)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	input.UserID = userID
	if err := handler.accountService.UpdatePreferences(request.Context(), &input); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, input)
}

// # Session Security Endpoints

/*
GET /api/v1/me/sessions.

Description: Enumerates all devices currently authenticated into the user's account.

Response:
  - 200: []SessionInfo: List of active device sessions
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) listSessions(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	sessions, err := handler.accountService.ListSessions(request.Context(), userID, "")
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, sessions)
}

/*
DELETE /api/v1/me/sessions/{id}.

Description: Forces a sign-out on a specific device identified by its session ID.

Request:
  - id: string (Session UUID)

Response:
  - 204: No Content: Session terminated successfully
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) revokeSession(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	sessionID := chi.URLParam(request, "id")

	if err := handler.accountService.RevokeSession(request.Context(), userID, sessionID); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}

/*
DELETE /api/v1/me/sessions.

Description: Forces a sign-out on all devices except the one making the request.

Response:
  - 204: No Content: All other sessions terminated
  - 401: ErrUnauthorized: Authentication required
*/
func (handler *Handler) revokeOtherSessions(writer http.ResponseWriter, request *http.Request) {
	userID, err := requestutil.RequiredUserID(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.accountService.RevokeOtherSessions(request.Context(), userID, ""); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.NoContent(writer)
}
