package journal

import (
	"context"
)

// Store is a collection of journals.
type Store interface {
	// Open returns the journal with the given name.
	Open(ctx context.Context, name string) (Journal, error)
}
