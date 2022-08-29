package queue_test

import (
	"context"

	. "github.com/dogmatiq/veracity/queue"
)

type journalStub struct {
	Journal

	WriteFunc func(
		ctx context.Context,
		offset uint64,
		e JournalEntry,
	) error
}

func (s *journalStub) Write(
	ctx context.Context,
	offset uint64,
	r JournalEntry,
) error {
	if s.WriteFunc != nil {
		return s.WriteFunc(ctx, offset, r)
	}

	return s.Journal.Write(ctx, offset, r)
}
