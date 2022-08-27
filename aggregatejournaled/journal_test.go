package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregatejournaled"
)

type journalStub struct {
}

func (s *journalStub) Read(
	ctx context.Context,
	hk, id string,
	offset uint64,
) ([]Record, error) {
	panic("not implemented")
}

func (s *journalStub) Write(
	ctx context.Context,
	hk, id string,
	offset uint64,
	r Record,
) error {
	panic("not implemented")
}

func (s *journalStub) Truncate(
	ctx context.Context,
	hk, id string,
	offset uint64,
) error {
	panic("not implemented")
}
