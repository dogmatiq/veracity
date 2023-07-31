package kv

import "context"

// A RangeFunc is a function used to range over the key/value pairs in a
// [Keyspace].
//
// If err is non-nil, ranging stops and err is propagated up the stack.
// Otherwise, if ok is false, ranging stops without any error being propagated.
type RangeFunc func(ctx context.Context, k, v []byte) (ok bool, err error)

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

	// Range invokes fn for each key in the keyspace in an undefined order.
	Range(ctx context.Context, fn RangeFunc) error

	// Close closes the keyspace.
	Close() error
}
