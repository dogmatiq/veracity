package kvstore

import "context"

// Store is an interface for a binary key/value store.
type Store interface {
	Set(ctx context.Context, k, v []byte) error
	Get(ctx context.Context, k []byte) ([]byte, error)
}
