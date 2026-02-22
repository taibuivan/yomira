// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package slug generates ASCII URL slugs from arbitrary Unicode strings.
//
// # Usage
//
// Slugs are used as human-readable identifiers for comics (e.g., "solo-leveling").
// This package handles normalization, accent removal, and character sanitization.
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	// nonAlphanumeric matches any sequence of non-alphanumeric, non-hyphen characters.
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)
	// multiHyphen collapses multiple consecutive hyphens into one.
	multiHyphen = regexp.MustCompile(`-{2,}`)
)

// From converts an arbitrary Unicode string into a URL-safe ASCII slug.
//
// # Transformation Pipeline
//
// 1. Normalizes to NFD (decomposes accented chars: é → e + combining acute).
// 2. Removes combining marks (accents).
// 3. Converts to lowercase.
// 4. Replaces non-alphanumeric characters with hyphens.
// 5. Collapses multiple hyphens and trims leading/trailing hyphens.
func From(s string) string {
	// 1. Normalize and remove accents
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn))
	result, _, _ := transform.String(t, s)

	// 2. Lowercase
	result = strings.ToLower(result)

	// 3. Replace whitespace and special chars with hyphens
	result = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, result)

	// 4. Clean up hyphenation
	result = nonAlphanumeric.ReplaceAllString(result, "-")
	result = multiHyphen.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	return result
}

// isMn reports whether r is a Unicode non-spacing mark (e.g., accents).
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}
