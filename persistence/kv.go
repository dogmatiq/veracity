package persistence

import "context"

// KeyValueStore is an interface for a simple key/value store.
type KeyValueStore interface {
	Set(ctx context.Context, k, v []byte) error
	Get(ctx context.Context, k []byte) ([]byte, error)
}
