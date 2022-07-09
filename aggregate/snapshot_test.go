package aggregate_test

import (
	"context"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/veracity/aggregate"
)

// snapshotReaderStub is a test implementation of the SnapshotReader interface.
type snapshotReaderStub struct {
	SnapshotReader

	ReadSnapshotFunc func(
		ctx context.Context,
		hk, id string,
		r dogma.AggregateRoot,
		minOffset uint64,
	) (snapshotOffset uint64, ok bool, _ error)
}

func (s *snapshotReaderStub) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minOffset uint64,
) (snapshotOffset uint64, ok bool, _ error) {
	if s.ReadSnapshotFunc != nil {
		return s.ReadSnapshotFunc(ctx, hk, id, r, minOffset)
	}

	if s.SnapshotReader != nil {
		return s.SnapshotReader.ReadSnapshot(ctx, hk, id, r, minOffset)
	}

	return 0, false, nil
}
