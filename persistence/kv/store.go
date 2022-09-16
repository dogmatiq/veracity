package kv

import (
	"context"
)

// Store is a collection of keyspaces.
//
// Each keyspace in the store is identified by a path, represented as a slice of
// string elements.
type Store interface {
	// Open returns the keyspace at the given path.
	//
	// The path uniquely identifies the keyspace. It must not be empty. Each
	// element must be a non-empty UTF-8 string consisting solely of printable
	// Unicode characters, excluding whitespace. A printable character is any
	// character from the Letter, Mark, Number, Punctuation or Symbol
	// categories.
	Open(ctx context.Context, path ...string) (Keyspace, error)
}
