package persistence

import "context"

// A KeyValueStore is a simple associative database that creates mappings
// between opaque binary keys and values.
type KeyValueStore interface {
	// Set associates the value v with the key k.
	//
	// Any value already associated with k is replaced.
	//
	// Setting v to an empty (or nil) slice deletes any existing association.
	Set(ctx context.Context, k, v []byte) error

	// Get returns the value associated with the key k.
	//
	// It returns an empty value if there is no value associated with k. There
	// is no distinction made between an empty slice and a nil slice.
	Get(ctx context.Context, k []byte) (v []byte, err error)
}
