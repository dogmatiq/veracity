package persistence

import "context"

// KeyValueStore is an interface for a simple key/value store.
type KeyValueStore interface {
	Set(ctx context.Context, k string, v []byte) error
	Get(ctx context.Context, k string) ([]byte, error)
}
