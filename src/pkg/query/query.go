package query

import (
	"strconv"
	"strings"
)

// IntSlice parses a slice of string values from URL query parameters
// into a slice of integers. Invalid entries are ignored safely.
func IntSlice(vals []string) []int {
	var res []int
	for _, v := range vals {
		if i, err := strconv.Atoi(v); err == nil {
			res = append(res, i)
		}
	}
	return res
}

// StringSlice parses a single comma-separated query string
// into a trimmed slice of strings.
func StringSlice(val string) []string {
	if val == "" {
		return nil
	}
	var res []string
	for _, v := range strings.Split(val, ",") {
		clean := strings.TrimSpace(v)
		if clean != "" {
			res = append(res, clean)
		}
	}
	return res
}
