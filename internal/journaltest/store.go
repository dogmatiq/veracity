package journaltest

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/journal"
)

// StoreStub is a test implementation of the journal.Store interface.
type StoreStub struct {
	journal.Store

	OpenFunc func(ctx context.Context, path ...string) (journal.Journal, error)
}

// Open returns the journal at the given path.
func (s *StoreStub) Open(ctx context.Context, path ...string) (journal.Journal, error) {
	if s.OpenFunc != nil {
		return s.OpenFunc(ctx, path...)
	}

	if s.Store != nil {
		return s.Store.Open(ctx, path...)
	}

	return nil, errors.New("<not implemented>")
}
