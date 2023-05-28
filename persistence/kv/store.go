package kv

import (
	"context"
)

// Store is a collection of keyspaces.
type Store interface {
	// Open returns the keyspace with the given name.
	Open(ctx context.Context, name string) (Keyspace, error)
}
