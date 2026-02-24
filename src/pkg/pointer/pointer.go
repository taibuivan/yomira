// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package pointer provides utilities for working with pointers in Go.

It heavily utilizes generics to simplify the creation and dereferencing
of pointers cleanly, avoiding boilerplate code in the application logic.

Key Functions:
  - To: Creates a pointer from a value literal.
  - Val: Safely dereferences a pointer, returning the zero value if nil.
  - Fallback: Safely dereferences a pointer, returning a fallback value if nil.
*/
package pointer

// To returns a pointer to the provided value.
// It is useful when you need to pass a primitive value to a function or struct field
// that expects a pointer (e.g. ptr.To("something")).
func To[T any](v T) *T {
	return &v
}

// Val safely dereferences a pointer.
// If the pointer is nil, it returns the zero value of the underlying type.
func Val[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Fallback safely dereferences a pointer.
// If the pointer is nil, it returns the provided fallback value instead.
func Fallback[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
