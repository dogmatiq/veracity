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
	) (uint64, bool, error)
}

func (s *snapshotReaderStub) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minOffset uint64,
) (uint64, bool, error) {
	if s.ReadSnapshotFunc != nil {
		return s.ReadSnapshotFunc(ctx, hk, id, r, minOffset)
	}

	if s.SnapshotReader != nil {
		return s.SnapshotReader.ReadSnapshot(ctx, hk, id, r, minOffset)
	}

	return 0, false, nil
}

// snapshotWriterStub is a test implementation of the SnapshotWriter interface.
type snapshotWriterStub struct {
	SnapshotWriter

	WriteSnapshotFunc func(
		ctx context.Context,
		hk, id string,
		r dogma.AggregateRoot,
		snapshotOffset uint64,
	) error
}

func (s *snapshotWriterStub) WriteSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	snapshotOffset uint64,
) error {
	if s.WriteSnapshotFunc != nil {
		return s.WriteSnapshotFunc(ctx, hk, id, r, snapshotOffset)
	}

	if s.SnapshotWriter != nil {
		return s.SnapshotWriter.WriteSnapshot(ctx, hk, id, r, snapshotOffset)
	}

	return nil
}
