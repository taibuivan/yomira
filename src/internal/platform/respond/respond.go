// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package respond provides a unified API response envelope for the platform.

It ensures that every HTTP response, whether a success payload or an error
diagnostic, follows a predictable JSON structure for client robustness.

Architecture:

  - Envelope: All responses are wrapped in a standard structure.
  - JSON: Default content-type is 'application/json; charset=utf-8'.
  - Errors: Integrates with 'apperr' for consistent error reporting.

This package eliminates the need for manual JSON marshalling in individual handlers.
*/
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/ctxutil"
	"github.com/taibuivan/yomira/pkg/pagination"
)

// # JSON Envelopes

// SuccessEnvelope is the JSON envelope for successful single-resource responses.
type SuccessEnvelope struct {
	Data interface{} `json:"data"`
}

// PaginatedEnvelope is the JSON envelope for paginated list responses.
type PaginatedEnvelope struct {
	Data interface{}     `json:"data"`
	Meta pagination.Meta `json:"meta"`
}

// ErrorEnvelope is the JSON envelope for error responses.
type ErrorEnvelope struct {
	Error   string              `json:"error"`
	Code    string              `json:"code"`
	Details []apperr.FieldError `json:"details,omitempty"`
}

// # Response Helpers

/*
JSON writes a JSON response with the given status code.

Parameters:
  - writer: http.ResponseWriter
  - statusCode: int (e.g., http.StatusOK)
  - payload: interface{} (Struct to encode)
*/
func JSON(writer http.ResponseWriter, statusCode int, payload interface{}) {

	// Set the common JSON header
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Write the status first
	writer.WriteHeader(statusCode)

	// Encode the payload directly to the stream
	_ = json.NewEncoder(writer).Encode(payload)
}

/*
OK writes a 200 OK response with data wrapped in the standard success envelope.

Parameters:
  - writer: http.ResponseWriter
  - data: interface{} (Business entity or DTO)
*/
func OK(writer http.ResponseWriter, data interface{}) {
	JSON(writer, http.StatusOK, SuccessEnvelope{Data: data})
}

/*
Created writes a 201 Created response with data wrapped in the standard success envelope.

Parameters:
  - writer: http.ResponseWriter
  - data: interface{} (The newly created object)
*/
func Created(writer http.ResponseWriter, data interface{}) {
	JSON(writer, http.StatusCreated, SuccessEnvelope{Data: data})
}

/*
Paginated writes a 200 OK response with paginated data and a metadata block.

Parameters:
  - writer: http.ResponseWriter
  - data: interface{} (Slice of items)
  - metadata: pagination.Meta (Page, Limit, Total)
*/
func Paginated(writer http.ResponseWriter, data interface{}, metadata pagination.Meta) {
	JSON(writer, http.StatusOK, PaginatedEnvelope{Data: data, Meta: metadata})
}

/*
NoContent writes a 204 No Content response.

Parameters:
  - writer: http.ResponseWriter
*/
func NoContent(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusNoContent)
}

/*
NotImplemented returns a placeholder 501 Not Implemented error.

Parameters:
  - writer: http.ResponseWriter
  - request: *http.Request
*/
func NotImplemented(writer http.ResponseWriter, request *http.Request) {
	Error(writer, request, &apperr.AppError{
		Code:       "NOT_IMPLEMENTED",
		Message:    "Endpoint is not fully implemented yet.",
		HTTPStatus: http.StatusNotImplemented,
	})
}

// # Error Handling

/*
Error converts any Go error into a standardized JSON API error response.

Description:
It inspects the error chain for [apperr.AppError]. If not found, it wraps it
as an Internal Server Error. High-severity errors (5xx) are automatically logged.

Parameters:
  - writer: http.ResponseWriter
  - request: *http.Request
  - err: error (Underlying failure)
*/
func Error(writer http.ResponseWriter, request *http.Request, err error) {
	var appError *apperr.AppError

	// If the error is not already an [apperr.AppError], wrap it as an Internal Server Error
	if !errors.As(err, &appError) {

		// Log the raw details internally for debugging
		logger := ctxutil.GetLogger(request.Context())
		logger.ErrorContext(request.Context(), "unhandled_error_swallowed",
			slog.String("error", err.Error()),
			slog.String("request_id", ctxutil.GetRequestID(request.Context())),
		)

		appError = apperr.Internal(err)
	}

	// Always log 5xx errors as they indicate server-side failures that need attention
	if appError.HTTPStatus >= 500 {
		logger := ctxutil.GetLogger(request.Context())
		logger.ErrorContext(request.Context(), "api_server_error",
			slog.String("code", appError.Code),
			slog.String("request_id", ctxutil.GetRequestID(request.Context())),
			slog.Any("cause", appError.Cause),
		)
	}

	// Write the final standardized JSON error payload
	JSON(writer, appError.HTTPStatus, ErrorEnvelope{
		Error:   appError.Message,
		Code:    appError.Code,
		Details: appError.Details,
	})
}
