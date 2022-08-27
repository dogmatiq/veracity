package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregatejournaled"
)

type journalStub struct {
	Journal
}

func (s *journalStub) Read(
	ctx context.Context,
	hk, id string,
	offset uint64,
) ([]Record, error) {
	return s.Journal.Read(ctx, hk, id, offset)
}

func (s *journalStub) Write(
	ctx context.Context,
	hk, id string,
	offset uint64,
	r Record,
) error {
	return s.Journal.Write(ctx, hk, id, offset, r)
}
