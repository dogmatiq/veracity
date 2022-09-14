package journaltest

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/journal"
)

type OpenerStub[R any] struct {
	journal.Opener[R]

	OpenFunc func(ctx context.Context, path ...string) (journal.Journal[R], error)
}

func (o *OpenerStub[R]) Open(ctx context.Context, path ...string) (journal.Journal[R], error) {
	if o.OpenFunc != nil {
		return o.OpenFunc(ctx, path...)
	}

	if o.Opener != nil {
		return o.Opener.Open(ctx, path...)
	}

	return nil, errors.New("<not implemented>")
}
