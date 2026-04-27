package common

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// Sorted iterates a map in ascending key order. Key type is any ordered type
// (strings, ints, etc.); value type is unconstrained.
func Sorted[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}

// Ptr returns a pointer to v. Use in place of smithy-go's ptr.String (which
// this project keeps for AWS-adjacent code) anywhere else a quick "address
// of literal" is needed — generic, so it works for any type.
func Ptr[T any](v T) *T {
	return &v
}
