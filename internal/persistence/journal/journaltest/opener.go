package journaltest

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/internal/persistence/journal"
)

// OpenerStub is a test implementation of the Opener[R] interface.
type OpenerStub[R any] struct {
	journal.Opener[R]

	OpenJournalFunc func(ctx context.Context, key string) (journal.Journal[R], error)
}

// OpenJournal opens the journal identified by the given key.
func (o *OpenerStub[R]) OpenJournal(
	ctx context.Context,
	key string,
) (journal.Journal[R], error) {
	if o.OpenJournalFunc != nil {
		return o.OpenJournalFunc(ctx, key)
	}

	if o.Opener != nil {
		return o.Opener.OpenJournal(ctx, key)
	}

	return nil, errors.New("<not implemented>")
}
