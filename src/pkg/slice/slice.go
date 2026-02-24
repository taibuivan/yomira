// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package slice compliments the standard [slices] package by providing functional
programming utilities (Map, Filter) leveraging generics.
*/
package slice

// Map maps a slice of type T to a slice of type U using the provided transformation function.
func Map[T any, U any](input []T, transform func(T) U) []U {
	if input == nil {
		return nil
	}

	result := make([]U, len(input))
	for i, v := range input {
		result[i] = transform(v)
	}

	return result
}

// Filter filters a slice, returning only elements where the predicate function evaluates to true.
func Filter[T any](input []T, predicate func(T) bool) []T {
	if input == nil {
		return nil
	}

	// Not pre-allocating to full length to avoid excessive memory on heavy filters
	var result []T
	for _, v := range input {
		if predicate(v) {
			result = append(result, v)
		}
	}

	return result
}

// Reduce reduces a slice into a single accumulated result using the reducer function.
func Reduce[T any, U any](input []T, initial U, reducer func(accumulator U, current T) U) U {
	result := initial
	for _, v := range input {
		result = reducer(result, v)
	}
	return result
}
