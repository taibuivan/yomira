// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package apperr defines the centralized error handling framework for Yomira.

It provides a rich error type that bridges the gap between low-level Domain/Storage
errors and high-level HTTP responses.

Architecture:

  - AppError: A struct containing machine-readable ErrorCode and user-friendly messages.
  - Localization: Support for translated error messages (if needed in the future).
  - Mapping: Explicit mapping from AppError to standard HTTP Status Codes.

Every error that leaves the service layer should be wrapped as an [AppError] to ensure
consistent API responses.
*/
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the canonical error type for the Yomira API.
//
// It carries an HTTP status code, a machine-readable code, a client-safe
// message, and an optional slice of field-level validation errors.
//
// # Security
//
// The Cause field is for server-side logging only and is never sent to clients
// to avoid leaking internal implementation details (e.g., SQL queries).
type AppError struct {
	// Code is a machine-readable error identifier (e.g. "NOT_FOUND", "CONFLICT").
	Code string `json:"code"`
	// Message is a human-readable description safe to return to the client.
	Message string `json:"error"`
	// HTTPStatus is the HTTP response status code.
	HTTPStatus int `json:"-"`
	// Cause is the underlying error, used for server-side logging only.
	Cause error `json:"-"`
	// Details holds per-field validation errors for VALIDATION_ERROR responses.
	Details []FieldError `json:"details,omitempty"`
}

// FieldError represents a single field-level validation failure.
type FieldError struct {
	// Field is the JSON field name that failed validation.
	Field string `json:"field"`
	// Message is the human-readable description of the failure.
	Message string `json:"message"`
}

// Error implements the error interface. It returns the client-safe message.
func (e *AppError) Error() string { return e.Message }

// Unwrap allows [errors.Is] and [errors.As] to traverse the cause chain.
func (e *AppError) Unwrap() error { return e.Cause }

// # Client Errors (4xx)

// NotFound creates a 404 [AppError] for a named resource.
//
// Example:
//
//	apperr.NotFound("Comic") // Returns "Comic not found"
func NotFound(resource string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    resource + " not found",
		HTTPStatus: http.StatusNotFound,
	}
}

// Unauthorized creates a 401 [AppError].
func Unauthorized(msg string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    msg,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// Forbidden creates a 403 [AppError].
func Forbidden(msg string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    msg,
		HTTPStatus: http.StatusForbidden,
	}
}

// Conflict creates a 409 [AppError] for duplicate or unique-constraint violations.
func Conflict(msg string) *AppError {
	return &AppError{
		Code:       "CONFLICT",
		Message:    msg,
		HTTPStatus: http.StatusConflict,
	}
}

// ValidationError creates a 400 [AppError] with optional per-field details.
func ValidationError(msg string, details ...FieldError) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    msg,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// RateLimited creates a 429 [AppError].
func RateLimited(retryAfterSeconds int) *AppError {
	return &AppError{
		Code:       "RATE_LIMITED",
		Message:    fmt.Sprintf("Too many requests. Try again in %ds.", retryAfterSeconds),
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// Unprocessable creates a 422 [AppError] for semantically invalid input.
func Unprocessable(msg string) *AppError {
	return &AppError{
		Code:       "UNPROCESSABLE",
		Message:    msg,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

// # Server Errors (5xx)

// Internal creates a 500 [AppError] wrapping an unexpected server-side error.
// The cause is stored for logging but is never sent to the client.
func Internal(cause error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// ServiceUnavailable creates a 503 [AppError] for maintenance mode.
func ServiceUnavailable(msg string) *AppError {
	return &AppError{
		Code:       "SERVICE_UNAVAILABLE",
		Message:    msg,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// # Helpers

// IsAppError reports whether err (or any error in its chain) is an [*AppError].
func IsAppError(err error) bool {
	var ae *AppError
	return errors.As(err, &ae)
}

// As extracts the [*AppError] from err's chain. It returns nil if not found.
func As(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}
