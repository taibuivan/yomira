// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package respond provides HTTP response helpers used by all API handlers.
//
// # Architecture
//
// This package centralizes the presentation logic for HTTP responses.
// It ensures that every response (Success or Error) across the entire application
// follows a strict, predictable JSON envelope structure. This consistency is
// crucial for mobile apps and frontend SPAs to parse data robustly.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/buivan/yomira/internal/platform/apperr"
	"github.com/buivan/yomira/internal/platform/ctxkey"
	"github.com/buivan/yomira/pkg/pagination"
)

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

// JSON writes a JSON response with the given status code.
func JSON(writer http.ResponseWriter, statusCode int, payload interface{}) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

// OK writes a 200 OK response with data wrapped in the standard success envelope.
func OK(writer http.ResponseWriter, data interface{}) {
	JSON(writer, http.StatusOK, SuccessEnvelope{Data: data})
}

// Created writes a 201 Created response with data wrapped in the standard success envelope.
func Created(writer http.ResponseWriter, data interface{}) {
	JSON(writer, http.StatusCreated, SuccessEnvelope{Data: data})
}

// Paginated writes a 200 OK response with paginated data and a metadata block.
func Paginated(writer http.ResponseWriter, data interface{}, metadata pagination.Meta) {
	JSON(writer, http.StatusOK, PaginatedEnvelope{Data: data, Meta: metadata})
}

// NoContent writes a 204 No Content response.
func NoContent(writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusNoContent)
}

// Error converts any Go error into a standardized JSON API error response.
func Error(writer http.ResponseWriter, request *http.Request, err error) {
	var appError *apperr.AppError
	if !errors.As(err, &appError) {
		// Unexpected internal error: log full details but hide them from the client for security.
		logger := getLoggerFromContext(request)
		logger.ErrorContext(request.Context(), "unhandled_error_swallowed",
			slog.String("error", err.Error()),
			slog.String("request_id", getRequestIDFromContext(request)),
		)
		appError = apperr.Internal(err)
	}

	// Always log 5xx errors as they indicate server-side issues.
	if appError.HTTPStatus >= 500 {
		logger := getLoggerFromContext(request)
		logger.ErrorContext(request.Context(), "api_server_error",
			slog.String("code", appError.Code),
			slog.String("request_id", getRequestIDFromContext(request)),
			slog.Any("cause", appError.Cause),
		)
	}

	JSON(writer, appError.HTTPStatus, ErrorEnvelope{
		Error:   appError.Message,
		Code:    appError.Code,
		Details: appError.Details,
	})
}

// getLoggerFromContext extracts the per-request logger.
func getLoggerFromContext(request *http.Request) *slog.Logger {
	if logger, ok := request.Context().Value(ctxkey.KeyLogger).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// getRequestIDFromContext extracts the X-Request-ID for log correlation.
func getRequestIDFromContext(request *http.Request) string {
	if id, ok := request.Context().Value(ctxkey.KeyRequestID).(string); ok {
		return id
	}
	return ""
}
