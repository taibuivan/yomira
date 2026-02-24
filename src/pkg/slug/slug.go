// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package slug generates SEO-friendly ASCII identifiers from Unicode strings.

It handles everything from accent removal (normalization) to character
sanitization, ensuring that titles like "Sólo Leveling" become "solo-leveling".

Transformation Pipeline:

 1. NFD Normalization: Decomposes accented chars (é -> e + accent).
 2. Accent Stripping: Removes combining marks.
 3. Lowercasing: Ensures URL uniformity.
 4. Sanitization: Replaces non-alphanumeric chars with hyphens.
 5. Clean-up: Collapses multiple hyphens and trims boundaries.

Slugs generated here are used as primary human-readable lookups for Comics.
*/
package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// # Common RegEx

var (
	// nonAlphanumeric matches any sequence of non-alphanumeric, non-hyphen characters.
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)

	// multiHyphen collapses multiple consecutive hyphens into one.
	multiHyphen = regexp.MustCompile(`-{2,}`)
)

// # Public API

// From converts an arbitrary Unicode string into a URL-safe ASCII slug.
func From(s string) string {

	// 1. Normalize and remove accents (e.g. "é" becomes "e")
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn))
	result, _, _ := transform.String(t, s)

	// 2. Convert to Lowercase for uniformity
	result = strings.ToLower(result)

	// 3. Replace non-standard characters with hyphens
	result = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, result)

	// 4. Final cleaning: collapse multiple hyphens and trim boundaries
	result = nonAlphanumeric.ReplaceAllString(result, "-")
	result = multiHyphen.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	return result
}

// # Internal Helpers

// isMn reports whether r is a Unicode non-spacing mark (e.g. accents).
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}
