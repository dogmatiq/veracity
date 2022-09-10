package journaltest

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/journal"
)

// OpenerStub is a test implementation of the Opener[R] interface.
type OpenerStub[R any] struct {
	journal.Opener[R]

	OpenFunc func(ctx context.Context, key string) (journal.Journal[R], error)
}

// Open opens the journal identified by the given key.
func (o *OpenerStub[R]) Open(ctx context.Context, key string) (journal.Journal[R], error) {
	if o.OpenFunc != nil {
		return o.OpenFunc(ctx, key)
	}

	if o.Opener != nil {
		return o.Opener.Open(ctx, key)
	}

	return nil, errors.New("<not implemented>")
}
