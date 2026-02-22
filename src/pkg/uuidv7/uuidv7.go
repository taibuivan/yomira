// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package uuidv7 wraps google/uuid to generate time-ordered UUIDv7 values.
//
// # Why UUIDv7?
//
// It is used as the primary key type across all Yomira tables. Because it is
// time-sortable, it ensures clustered-index friendliness in PostgreSQL,
// preventing the "index fragmentation" common with random UUIDv4.
package uuidv7

import "github.com/google/uuid"

// New generates a new UUIDv7 string.
//
// # Safety
//
// It panics only if the OS random source is unavailable (extremely rare).
// This is acceptable as OS entropy failure is an unrecoverable system-level error.
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic("uuidv7: failed to generate UUID: " + err.Error())
	}

	return id.String()
}

// Must generates a new UUIDv7 or panics.
//
// This is an alias for [New] kept for readability and consistency with
// Go's "Must" pattern in call sites.
func Must() string {
	return New()
}
