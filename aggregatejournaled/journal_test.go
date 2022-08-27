package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregatejournaled"
)

type journalStub struct {
	Journal

	WriteFunc func(
		ctx context.Context,
		hk, id string,
		offset uint64,
		e JournalEntry,
	) error
}

func (s *journalStub) Read(
	ctx context.Context,
	hk, id string,
	offset uint64,
) ([]JournalEntry, error) {
	return s.Journal.Read(ctx, hk, id, offset)
}

func (s *journalStub) Write(
	ctx context.Context,
	hk, id string,
	offset uint64,
	r JournalEntry,
) error {
	if s.WriteFunc != nil {
		return s.WriteFunc(ctx, hk, id, offset, r)
	}

	return s.Journal.Write(ctx, hk, id, offset, r)
}
