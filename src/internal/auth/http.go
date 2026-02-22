// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package handler contains the HTTP delivery layer for domain use cases.
//
// # Architecture
//
// Handlers act as the "gatekeepers" to the system. They are responsible for:
//   - JSON Request parsing and strict input validation.
//   - Mapping HTTP contexts to service layer method calls.
//   - Standardizing JSON response formats via the [respond] package.
//
// They contain NO business logic or database queries.
package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/buivan/yomira/internal/platform/respond"
	"github.com/buivan/yomira/internal/platform/validate"
)

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

	router.Post("/register", handler.register)
	router.Post("/login", handler.login)

	return router
}

// registerRequest represents the JSON payload expected for account creation.
type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// register handles POST /api/v1/auth/register requests.
//
// # Parameters
//   - writer: The HTTP response constructor.
//   - request: The incoming HTTP request payload.
//
// # Returns
//   - Writes HTTP 201 Created on success with the User profile.
//   - Writes HTTP 400 Bad Request if validation rules fail.
//   - Writes HTTP 409 Conflict if email/username is taken.
func (handler *Handler) register(writer http.ResponseWriter, request *http.Request) {
	// ── 1. Payload Extraction ─────────────────────────────────────────────

	var input registerRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	// ── 2. Boundary Validation (Explicit & Mandatory) ────────────────────

	// Prevent malformed data from reaching the service layer.
	// We use the validate helper to ensure consistent ErrorEnvelope shapes.
	if input.Username == "" || len(input.Username) < 3 {
		respond.Error(writer, request, validate.RequiredError("username", "must be at least 3 characters"))
		return
	}
	if input.Email == "" {
		// Proper Regex email validation is done inside the service/value object
		// or validator chain, this is a fast-fail check.
		respond.Error(writer, request, validate.RequiredError("email", "is required"))
		return
	}
	if input.Password == "" || len(input.Password) < 8 {
		respond.Error(writer, request, validate.RequiredError("password", "must be at least 8 characters"))
		return
	}

	// ── 3. Application Execution ──────────────────────────────────────────

	user, err := handler.authService.Register(request.Context(), RegisterInput{
		Username:    input.Username,
		Email:       input.Email,
		Password:    input.Password,
		DisplayName: input.DisplayName,
	})

	// Service handles uniqueness checks and Bcrypt hashing.
	// If it fails, we simply pass the domain error to the respond helper
	// which will automatically map it to the correct HTTP status code.
	if err != nil {
		respond.Error(writer, request, err)
		return
	}

	// ── 4. Presentation Output ────────────────────────────────────────────

	respond.Created(writer, user)
}

// loginRequest represents the JSON payload expected for authentication.
type loginRequest struct {
	Login    string `json:"login"` // Can be Username or Email
	Password string `json:"password"`
}

// login handles POST /api/v1/auth/login requests.
//
// # Parameters
//   - writer: The HTTP response constructor.
//   - request: The incoming HTTP request payload.
//
// # Returns
//   - Writes HTTP 200 OK on success with AccessToken and User profile.
//   - Writes HTTP 401 Unauthorized for bad credentials.
func (handler *Handler) login(writer http.ResponseWriter, request *http.Request) {
	// ── 1. Payload Extraction ─────────────────────────────────────────────

	var input loginRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		respond.Error(writer, request, validate.ErrInvalidJSON)
		return
	}

	// ── 2. Boundary Validation ────────────────────────────────────────────

	if input.Login == "" || input.Password == "" {
		respond.Error(writer, request, validate.RequiredError("login/password", "are required"))
		return
	}

	// ── 3. Application Execution ──────────────────────────────────────────

	session, err := handler.authService.Login(request.Context(), LoginInput{
		Login:    input.Login,
		Password: input.Password,
	})

	if err != nil {
		// Will return HTTP 401 Unauthorzed without leaking reason (e.g. wrong pass vs wrong email)
		respond.Error(writer, request, err)
		return
	}

	// ── 4. Presentation Output ────────────────────────────────────────────

	respond.OK(writer, map[string]any{
		"access_token": session.AccessToken,
		"user":         session.User,
	})
}
