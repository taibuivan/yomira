// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package pagination provides standardized navigation for collection-based APIs.

It handles everything from parsing 'page' and 'limit' query parameters to
calculating total pages and offsets for database queries.

Usage:

	params := pagination.FromRequest(request)
	meta := pagination.NewMeta(params.Page, params.Limit, totalCount)

Architecture:

  - Request: Standardizes query parameter names and boundary clamping.
  - Response: Provides a uniform 'Meta' object for frontend hydration.
  - Safety: Prevents system abuse by enforcing MaxLimit.

This package ensures a consistent user experience for all list-based endpoints.
*/
package pagination

import (
	"net/http"

	"github.com/taibuivan/yomira/pkg/convert"
)

// # Common Defaults

const (
	// DefaultLimit is the number of items per page if not specified.
	DefaultLimit = 20

	// MaxLimit is the upper bound for items per page to prevent system abuse.
	MaxLimit = 100

	// DefaultPage is the starting page (1-indexed).
	DefaultPage = 1
)

// # Request Parameters

// Params holds the parsed page and limit from a request's query string.
type Params struct {
	Page  int
	Limit int
}

// Offset returns the SQL OFFSET value derived from [Page] and [Limit].
func (params Params) Offset() int {

	// Ensure we don't return negative offsets
	if params.Page <= 1 {
		return 0
	}

	// Calculate the offset
	return (params.Page - 1) * params.Limit
}

// # Response Metadata

// Meta is the pagination metadata included in API list responses.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewMeta constructs pagination metadata for a response.
func NewMeta(page, limit, total int) Meta {

	// Calculate the total number of pages (rounding up)
	totalPages := 0

	// Ensure we don't return negative page counts
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	// Return the pagination metadata
	return Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// # Parsing Logic

// FromRequest parses "page" and "limit" query parameters from an HTTP request.
func FromRequest(request *http.Request) Params {

	// Parse individual parameters with base defaults
	page := parseIntParam(request, "page", DefaultPage)
	limit := parseIntParam(request, "limit", DefaultLimit)

	// Clamp the page to valid range
	if page < 1 {
		page = DefaultPage
	}

	// Clamp the limit to prevent resource exhaustion
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}

	// Return the parsed parameters
	return Params{Page: page, Limit: limit}
}

// parseIntParam parses a single integer query parameter with a fallback default.
func parseIntParam(request *http.Request, key string, defaultVal int) int {
	return convert.ToIntD(request.URL.Query().Get(key), defaultVal)
}
