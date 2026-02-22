// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package pagination provides shared types and helpers for API list endpoints.
//
// # Overview
//
// It standardizes how page-based navigation is requested via query parameters
// and how the resulting metadata is delivered in the API response envelope.
package pagination

import (
	"net/http"
	"strconv"
)

const (
	// DefaultLimit is the number of items per page if not specified.
	DefaultLimit = 20
	// MaxLimit is the upper bound for items per page to prevent system abuse.
	MaxLimit = 100
	// DefaultPage is the starting page (1-indexed).
	DefaultPage = 1
)

// Params holds the parsed page and limit from a request's query string.
type Params struct {
	Page  int
	Limit int
}

// Offset returns the SQL OFFSET value derived from [Page] and [Limit].
func (p Params) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// Meta is the pagination metadata included in API list responses.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewMeta constructs pagination metadata for a response.
//
// It automatically calculates the TotalPages based on the total count and limit.
func NewMeta(page, limit, total int) Meta {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// FromRequest parses "page" and "limit" query parameters from an HTTP request.
//
// # Clamping
//
// Invalid, negative, or excessive values are automatically clamped to
// [DefaultPage], [DefaultLimit], or [MaxLimit].
func FromRequest(r *http.Request) Params {
	page := parseIntParam(r, "page", DefaultPage)
	limit := parseIntParam(r, "limit", DefaultLimit)

	if page < 1 {
		page = DefaultPage
	}

	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}

	return Params{Page: page, Limit: limit}
}

// parseIntParam parses a single integer query parameter with a fallback default.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal
	}

	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}

	return n
}
