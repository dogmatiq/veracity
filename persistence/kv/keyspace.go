package kv

import "context"

// A Keyspace is an isolated collection of key/value pairs.
type Keyspace interface {
	// Get returns the value associated with k.
	//
	// If the key does not exist v is empty.
	Get(ctx context.Context, k []byte) (v []byte, err error)

	// Has returns true if k is present in the keyspace.
	Has(ctx context.Context, k []byte) (ok bool, err error)

	// Set associates a value with k.
	//
	// If v is empty, the key is deleted.
	Set(ctx context.Context, k, v []byte) error

	// Close closes the keyspace.
	Close() error
}
