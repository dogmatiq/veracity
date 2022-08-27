package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregatejournaled"
)

type snapshotStoreStub struct {
}

func (s *snapshotStoreStub) Read(
	ctx context.Context,
	hk, id string,
) (Snapshot, error) {
	panic("not implemented")
}

func (s *snapshotStoreStub) Write(
	ctx context.Context,
	hk, id string,
	sn Snapshot,
) error {
	panic("not implemented")
}
