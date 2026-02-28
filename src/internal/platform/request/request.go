// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package request provides utilities for extracting data from HTTP requests.

It abstracts away the underlying router's parameter extraction and common
body decoding patterns, ensuring consistent error handling and type safety.
*/
package requestutil

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/ctxutil"
	"github.com/taibuivan/yomira/internal/platform/sec"
	"github.com/taibuivan/yomira/internal/platform/validate"
)

/*
DecodeJSON reads the request body and decodes it into the target structure.

Parameters:
  - request: *http.Request
  - target: interface{} (Pointer to the destination struct)

Returns:
  - error: validate.ErrInvalidJSON if decoding fails, otherwise nil
*/
func DecodeJSON(request *http.Request, target interface{}) error {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		return validate.ErrInvalidJSON
	}
	return nil
}

/*
ID retrieves a named URL parameter (UUID/Slug) from the request.
*/
func ID(request *http.Request, name string) string {
	return chi.URLParam(request, name)
}

/*
Param retrieves a named URL parameter from the request.
*/
func Param(request *http.Request, name string) string {
	return chi.URLParam(request, name)
}

/*
Claims extracts the authenticated user claims from the request context.

Returns nil if the request is not authenticated.
*/
func Claims(request *http.Request) *sec.AuthClaims {
	return ctxutil.GetAuthUser(request.Context())
}

/*
RequiredClaims ensures the request is authenticated and returns the user claims.

Returns:
  - *sec.AuthClaims: The authenticated user claims
  - error: apperr.Unauthorized if the request is not authenticated
*/
func RequiredClaims(request *http.Request) (*sec.AuthClaims, error) {

	// Get user claims
	claims := ctxutil.GetAuthUser(request.Context())

	// If the user is not authenticated, return an error
	if claims == nil {
		return nil, apperr.Unauthorized("Authentication required")
	}

	return claims, nil
}

/*
RequiredUserID returns the User ID of the currently logged-in user.

Returns:
  - string: User UUID
  - error: apperr.Unauthorized if not authenticated
*/
func RequiredUserID(request *http.Request) (string, error) {

	// Get user claims
	claims, err := RequiredClaims(request)

	// If the user is not authenticated, return an error
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}
