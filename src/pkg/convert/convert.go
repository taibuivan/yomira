// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package convert provides quick type-conversion utilities.

It wraps standards like [strconv] to provide fault-tolerant conversions
(e.g., returning 0 instead of an error string parsing fails). This is highly
useful in API handler contexts parsing query parameters.

Do not use this package if distinguishing between malformed data and zero values
is important in your domain logic; use explicit standard libraries instead.
*/
package convert

import (
	"strconv"
)

// ToInt converts a string to an integer, silencing parsing errors.
// It returns 0 if the string is empty or cannot be parsed.
func ToInt(s string) int {

	// If the string is empty, return 0
	if s == "" {
		return 0
	}

	// Try to parse the string as an integer
	v, _ := strconv.Atoi(s)
	return v
}

// ToIntD converts a string to an int, returning the provided default if parsing fails or string is empty.
func ToIntD(str string, def int) int {

	// If the string is empty, return the default value
	if str == "" {
		return def
	}

	// Try to parse the string as an integer
	if v, err := strconv.Atoi(str); err == nil {
		return v
	}

	// If parsing fails, return the default value
	return def
}

// ToBool parses a boolean string ("true", "1", "false", "0").
// It returns false on empty string or parse error.
func ToBool(s string) bool {

	// If the string is empty, return false
	if s == "" {
		return false
	}

	// Try to parse the string as a boolean
	v, _ := strconv.ParseBool(s)
	return v
}

// ToFloat64 converts a string to a float64, swallowing errors.
func ToFloat64(s string) float64 {

	// If the string is empty, return 0
	if s == "" {
		return 0
	}

	// Try to parse the string as a float64
	v, _ := strconv.ParseFloat(s, 64)

	// If parsing fails, return 0
	return v
}
