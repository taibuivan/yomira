// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package dberr provides a bridge between low-level database errors and
// higher-level application errors.
package dberr

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/taibuivan/yomira/internal/platform/apperr"
)

var (
	// ErrNotFound is a standard error returned when a queried row doesn't exist.
	ErrNotFound = apperr.NotFound("Resource")
)

// Wrap inspects a database error and wraps it into a meaningful [apperr.AppError].
// It hides internal database details from the client while classifying the error type.
func Wrap(err error, action string) error {
	if err == nil {
		return nil
	}

	// 1. Not Found mapping
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	// 2. Unknown query errors become Internal Server Errors
	// Real implementation would also check the Postgres SQLSTATE (e.g. 23505 for unique violation)
	return apperr.Internal(err)
}
