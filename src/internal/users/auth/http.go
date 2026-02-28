// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package auth provides the HTTP delivery layer for user identity management.

It implements the gateway for the authentication lifecycle—from account creation
to session management and recovery.

# Architecture

The handler acts as a thin mediation layer between the web and domain services:
  - Protocol: Standard RESTful JSON interface.
  - Security: Handles JWT orchestration and refresh token cookie injection.
  - Verification: Enforces strict input validation before passing to [Service].

This layer is strictly responsible for transport concerns (status codes, headers, JSON).
*/
package auth

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/constants"
	"github.com/taibuivan/yomira/internal/platform/middleware"
	requestutil "github.com/taibuivan/yomira/internal/platform/request"
	"github.com/taibuivan/yomira/internal/platform/respond"
	"github.com/taibuivan/yomira/internal/platform/validate"
)

// # Definitions & Constructors

// Handler implements authentication-related HTTP endpoints.
//
// # Scope
//
// This handler manages everything related to the user lifecycle entry points
// (Registration, Login, Password Reset callbacks).
type Handler struct {
	authService *Service
}

// NewHandler constructs a new [Handler] with its service dependency.
func NewHandler(service *Service) *Handler {
	return &Handler{authService: service}
}

// Routes returns a [chi.Router] configured with authentication-specific routes.
//
// # Endpoints
//   - POST /register : Creates a new account.
//   - POST /login    : Authenticates and returns a JWT.
func (handler *Handler) Routes() chi.Router {
	router := chi.NewRouter()

	// Public endpoints
	router.Post("/register", handler.register)
	router.Post("/login", handler.login)
	router.Post("/refresh", handler.refresh)
	router.Post("/verify-email", handler.verifyEmail)
	router.Post("/forgot-password", handler.forgotPassword)
	router.Post("/reset-password", handler.resetPassword)

	// Protected endpoints
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/logout", handler.logout)
		r.Post("/change-password", handler.changePassword)
	})

	return router
}

// # Request Payloads

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

/*
Register handles the creation of a new user account.

POST /api/v1/auth/register

Description: Validates input, checks for identity conflicts, and persists
a new user profile to the database.

Request:
  - Body: registerRequest (Username, Email, Password, DisplayName)

Response:
  - 201: User: Created user profile
  - 400: ErrInvalidJSON: Bad input or validation failure
  - 409: ErrConflict: Username or Email already exists
*/
func (handler *Handler) register(writer http.ResponseWriter, request *http.Request) {
	var input registerRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	validator := &validate.Validator{}
	validator.Required(FieldUsername, input.Username).
		MinLen(FieldUsername, input.Username, 3).
		Required(FieldEmail, input.Email).
		Email(FieldEmail, input.Email).
		Required(FieldPassword, input.Password).
		MinLen(FieldPassword, input.Password, 8)

	if err := validator.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	user, err := handler.authService.Register(request.Context(), RegisterInput{
		Username:    input.Username,
		Email:       input.Email,
		Password:    input.Password,
		DisplayName: input.DisplayName,
	})

	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.Created(writer, user)
}

/*
Login authenticates a user and establishes a session.

POST /api/v1/auth/login

Description: Verifies credentials, generates JWT access tokens, and injects
a secure refresh token cookie into the response.

Request:
  - Body: loginRequest (Login, Password)

Response:
  - 200: Session: Access token and User profile
  - 401: ErrUnauthorized: Invalid credentials or account locked
*/
func (handler *Handler) login(writer http.ResponseWriter, request *http.Request) {
	var input loginRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	validator := &validate.Validator{}
	validator.Required(FieldLogin, input.Login)
	validator.Required(FieldPassword, input.Password)

	if err := validator.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	session, err := handler.authService.Login(request.Context(), LoginInput{
		Login:     input.Login,
		Password:  input.Password,
		UserAgent: request.UserAgent(),
		IPAddress: getClientIP(request),
	})
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     constants.RefreshTokenCookieName,
		Value:    session.RefreshToken,
		Path:     constants.RefreshTokenCookiePath,
		Expires:  session.RefreshTokenExpiresAt,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	respond.OK(writer, map[string]any{
		"access_token": session.AccessToken,
		"user":         session.User,
	})
}

/*
Logout terminates the current user session.

POST /api/v1/auth/logout

Description: Invalidates the refresh token (if present) and clears the
security cookies from the client.

Response:
  - 204: No Content: Session terminated
*/
func (handler *Handler) logout(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(constants.RefreshTokenCookieName)

	if err == nil && cookie != nil && cookie.Value != "" {
		_ = handler.authService.Logout(request.Context(), cookie.Value)
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     constants.RefreshTokenCookieName,
		Value:    "",
		Path:     constants.RefreshTokenCookiePath,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	respond.NoContent(writer)
}

/*
Refresh issues a new access token using a valid refresh token.

POST /api/v1/auth/refresh

Description: Rotates the session by validating the refresh token cookie
and issuing a fresh access token and an updated refresh token.

Response:
  - 200: RefreshResponse: New access token credentials
  - 401: ErrUnauthorized: Missing or invalid refresh token
*/
func (handler *Handler) refresh(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(constants.RefreshTokenCookieName)
	if err != nil || cookie.Value == "" {
		respond.Error(writer, request, apperr.Unauthorized("Missing refresh token in cookies"))
		return
	}

	session, err := handler.authService.RefreshSession(
		request.Context(),
		cookie.Value,
		request.UserAgent(),
		getClientIP(request),
	)

	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     constants.RefreshTokenCookieName,
		Value:    session.RefreshToken,
		Path:     constants.RefreshTokenCookiePath,
		Expires:  session.RefreshTokenExpiresAt,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	respond.OK(writer, map[string]any{
		FieldAccessToken: session.AccessToken,
		FieldTokenType:   "Bearer",
		FieldExpiresIn:   AccessTokenTTL / time.Second,
	})
}

// getClientIP tries to extract the real IP address of a user over proxy environments.
func getClientIP(request *http.Request) string {

	ip := request.Header.Get("X-Real-IP")
	if ip == "" {
		ip = request.Header.Get("X-Forwarded-For")
	}

	if ip == "" {
		ip = request.RemoteAddr
	}
	return ip
}

/*
VerifyEmail confirms a user's email ownership.

POST /api/v1/auth/verify-email

Description: Validates an email verification token and marks the account as verified.

Request:
  - Body: verifyEmailRequest (Token)

Response:
  - 200: Success: Email verified
  - 400: ErrInvalidJSON: Missing or invalid token
*/
func (handler *Handler) verifyEmail(writer http.ResponseWriter, request *http.Request) {
	var input verifyEmailRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	if input.Token == "" {
		respond.Error(writer, request, validate.RequiredError(FieldToken, "is required"))
		return
	}

	if err := handler.authService.VerifyEmail(request.Context(), input.Token); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]string{
		FieldMessage: "Email verified successfully",
	})
}

/*
ForgotPassword initiates the password recovery flow.

POST /api/v1/auth/forgot-password

Description: Sends a password reset link to the provided email if the account exists.

Request:
  - Body: forgotPasswordRequest (Email)

Response:
  - 200: Success: Reset link sent (or generic security message)
  - 400: ErrInvalidJSON: Invalid email format
*/
func (handler *Handler) forgotPassword(writer http.ResponseWriter, request *http.Request) {
	var input forgotPasswordRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	v := &validate.Validator{}
	v.Required(FieldEmail, input.Email).Email(FieldEmail, input.Email)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	_, err := handler.authService.RequestPasswordReset(request.Context(), input.Email)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]string{
		FieldMessage: "If this email is registered, a reset link has been sent.",
	})
}

/*
ResetPassword completes the password recovery flow.

POST /api/v1/auth/reset-password

Description: Validates the reset token and updates the user's password.

Request:
  - Body: resetPasswordRequest (Token, Password)

Response:
  - 200: Success: Password updated
  - 400: ErrInvalidJSON: Bad token or weak password
*/
func (handler *Handler) resetPassword(writer http.ResponseWriter, request *http.Request) {
	var input resetPasswordRequest

	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	v := &validate.Validator{}
	v.Required(FieldToken, input.Token).
		Required(FieldPassword, input.Password).
		MinLen(FieldPassword, input.Password, 8)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	if err := handler.authService.ResetPassword(request.Context(), input.Token, input.Password); err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]string{
		FieldMessage: "Password updated successfully",
	})
}

/*
ChangePassword updates the authenticated user's password.

POST /api/v1/auth/change-password

Description: Verifies the current password and security context before
applying a new password.

Request:
  - Body: changePasswordRequest (CurrentPassword, NewPassword)

Response:
  - 200: Success: Password changed
  - 401: ErrUnauthorized: Session invalid or authentication required
  - 400: ErrInvalidJSON: Weak password or validation failure
*/
func (handler *Handler) changePassword(writer http.ResponseWriter, request *http.Request) {
	claims, err := requestutil.RequiredClaims(request)
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	cookie, err := request.Cookie(constants.RefreshTokenCookieName)
	if err != nil || cookie.Value == "" {
		respond.Error(writer, request, apperr.Unauthorized("Missing active session cookie"))
		return
	}

	var input changePasswordRequest
	if err := requestutil.DecodeJSON(request, &input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	v := &validate.Validator{}
	v.Required(FieldCurrentPassword, input.CurrentPassword).
		Required(FieldNewPassword, input.NewPassword).
		MinLen(FieldNewPassword, input.NewPassword, 8)

	if err := v.Err(); err != nil {
		respond.Error(writer, request, err)
		return
	}

	err = handler.authService.ChangePassword(
		request.Context(),
		claims.UserID,
		input.CurrentPassword,
		input.NewPassword,
		cookie.Value,
	)

	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	respond.OK(writer, map[string]string{
		FieldMessage: "Password changed successfully",
	})
}
