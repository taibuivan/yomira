// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package validate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/taibuivan/yomira/internal/platform/apperr"
	"github.com/taibuivan/yomira/internal/platform/validate"
)

/*
TestValidator_Required tests the mandatory field validation logic.
*/
func TestValidator_Required(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		hasError bool
	}{
		{"valid_string", "name", "Yomira", false},
		{"empty_string", "name", "", true},
		{"whitespace_only", "name", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &validate.Validator{}
			v.Required(tt.field, tt.value)

			if tt.hasError {
				assert.True(t, v.HasErrors())
				err := v.Err()
				require.NotNil(t, err)

				ae := apperr.As(err)
				require.NotNil(t, ae)
				assert.Equal(t, "VALIDATION_ERROR", ae.Code)
				assert.Equal(t, tt.field, ae.Details[0].Field)
			} else {
				assert.False(t, v.HasErrors())
				assert.Nil(t, v.Err())
			}
		})
	}
}

/*
TestValidator_Email checks the email format validation rule.
*/
func TestValidator_Email(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		isValid bool
	}{
		{"valid_email", "test@example.com", true},
		{"invalid_format", "invalid-email", false},
		{"missing_domain", "test@", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &validate.Validator{}
			v.Email("email", tt.email)

			if tt.isValid {
				assert.False(t, v.HasErrors())
			} else {
				assert.True(t, v.HasErrors())
			}
		})
	}
}

/*
TestValidator_Chain tests the fluent API (chaining multiple rules).
*/
func TestValidator_Chain(t *testing.T) {
	v := &validate.Validator{}

	// Multi-rule validation
	err := v.
		Required("username", "tai").
		MinLen("username", "tai", 3).
		MaxLen("username", "tai", 10).
		Email("email", "tai@yomira.com").
		Err()

	assert.NoError(t, err)
	assert.False(t, v.HasErrors())
}

/*
TestValidator_Chain_Failure tests error accumulation in the chain.
*/
func TestValidator_Chain_Failure(t *testing.T) {
	v := &validate.Validator{}

	err := v.
		Required("username", "").       // Fails
		MinLen("username", "a", 5).     // Fails
		Email("email", "not-an-email"). // Fails
		Err()

	require.Error(t, err)
	ae := apperr.As(err)
	require.NotNil(t, ae)

	// Should accumulate all 3 errors
	assert.Len(t, ae.Details, 3)
}
