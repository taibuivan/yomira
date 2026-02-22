// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package validate provides a chainable Validator that collects field-level
// errors before returning a single [apperr.AppError].
//
// # Architecture
//
// This package is used exclusively in the service layer — never in handlers or
// storage. It ensures that business logic only operates on semantically valid data.
package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/taibuivan/yomira/internal/platform/apperr"
)

var (
	// slugRegex matches slug format: lowercase letters, digits, hyphens.
	slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// uuidRegex matches a UUIDv4 or UUIDv7 string.
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// ErrInvalidJSON is returned when the request body cannot be decoded.
	ErrInvalidJSON = apperr.ValidationError("Invalid JSON payload")
)

// Validator collects field-level validation errors via a fluent, chainable API.
//
// # Concurrency
//
// Validator is not safe for concurrent use. A new instance must be created
// for every request/operation.
type Validator struct {
	errs []apperr.FieldError
}

// Required fails if the trimmed value is empty.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.add(field, "This field is required")
	}
	return v
}

// MaxLen fails if the Unicode character count exceeds max.
func (v *Validator) MaxLen(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.add(field, fmt.Sprintf("Maximum %d characters", max))
	}
	return v
}

// MinLen fails if the Unicode character count is below min.
func (v *Validator) MinLen(field, value string, min int) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.add(field, fmt.Sprintf("Minimum %d characters", min))
	}
	return v
}

// Range fails if the value is outside the [min, max] range (inclusive).
func (v *Validator) Range(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.add(field, fmt.Sprintf("Must be between %d and %d", min, max))
	}
	return v
}

// Email fails if the value is not a valid RFC 5322 email address.
func (v *Validator) Email(field, value string) *Validator {
	if _, err := mail.ParseAddress(value); err != nil {
		v.add(field, "Must be a valid email address")
	}
	return v
}

// Slug fails if the value is not a valid URL slug.
//
// # Format
//
// Slugs must consist only of lowercase letters, digits, and hyphens,
// with no leading or trailing hyphens.
func (v *Validator) Slug(field, value string) *Validator {
	if !slugRegex.MatchString(value) {
		v.add(field, "Must be a valid URL slug (lowercase letters, digits, hyphens only)")
	}
	return v
}

// UUID fails if the value is not a valid UUID string (case-insensitive).
func (v *Validator) UUID(field, value string) *Validator {
	lower := strings.ToLower(value)
	if !uuidRegex.MatchString(lower) {
		v.add(field, "Must be a valid UUID")
	}
	return v
}

// OneOf fails if the value is not in the allowed set of strings.
func (v *Validator) OneOf(field, value string, allowed ...string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.add(field, fmt.Sprintf("Must be one of: %s", strings.Join(allowed, ", ")))
	return v
}

// Custom adds a failure with a custom message if the condition is true.
//
// # Example
//
//	v.Custom ("score", score < 1 || score > 10, "Must be between 1 and 10")
func (v *Validator) Custom(field string, failed bool, message string) *Validator {
	if failed {
		v.add(field, message)
	}
	return v
}

// Err returns a [apperr.AppError] (VALIDATION_ERROR) if any rules failed,
// or nil if all rules passed.
//
// This is the only output method — call it at the end of the chain.
func (v *Validator) Err() error {
	if len(v.errs) == 0 {
		return nil
	}
	return apperr.ValidationError("Validation failed", v.errs...)
}

// HasErrors reports whether any validation rule has failed so far.
func (v *Validator) HasErrors() bool {
	return len(v.errs) > 0
}

// add appends a [apperr.FieldError] to the internal slice.
func (v *Validator) add(field, message string) {
	v.errs = append(v.errs, apperr.FieldError{Field: field, Message: message})
}

// RequiredError is a shortcut to create a single-field validation error.
func RequiredError(field, message string) *apperr.AppError {
	return apperr.ValidationError("Validation failed", apperr.FieldError{
		Field:   field,
		Message: message,
	})
}
