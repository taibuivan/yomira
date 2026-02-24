// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package uuidv7 provides time-ordered unique identifiers for the platform.

It wraps the standard UUID library to specifically generate Version 7 values,
which are optimized for database performance.

Advantages:

  - Sortable: Naturally ordered by creation time (millisecond precision).
  - Friendly: Prevents index fragmentation in PostgreSQL (B-tree optimal).
  - Compact: 128-bit storage, compatible with standard 'uuid' types.

This is the mandatory ID type for all primary keys in the Yomira ecosystem.
*/
package uuid

import "github.com/google/uuid"

// # Generators

// New generates a new UUIDv7 string.
func New() string {

	// Create a new version 7 UUID (time-sortable)
	id, err := uuid.NewV7()

	// entropy failure is an unrecoverable system-level error
	if err != nil {
		panic("uuidv7: failed to generate UUID: " + err.Error())
	}

	// Convert the UUID to a string
	return id.String()
}

// Must generates a new UUIDv7 or panics.
// Standard Go pattern for initialization where failure is not an option.
func Must() string {
	return New()
}
