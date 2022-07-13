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
		minRev uint64,
	) (uint64, bool, error)
}

func (s *snapshotReaderStub) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minRev uint64,
) (uint64, bool, error) {
	if s.ReadSnapshotFunc != nil {
		return s.ReadSnapshotFunc(ctx, hk, id, r, minRev)
	}

	if s.SnapshotReader != nil {
		return s.SnapshotReader.ReadSnapshot(ctx, hk, id, r, minRev)
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
		rev uint64,
	) error

	ArchiveSnapshotsFunc func(
		ctx context.Context,
		hk, id string,
	) error
}

func (s *snapshotWriterStub) WriteSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	rev uint64,
) error {
	if s.WriteSnapshotFunc != nil {
		return s.WriteSnapshotFunc(ctx, hk, id, r, rev)
	}

	if s.SnapshotWriter != nil {
		return s.SnapshotWriter.WriteSnapshot(ctx, hk, id, r, rev)
	}

	return nil
}

func (s *snapshotWriterStub) ArchiveSnapshots(
	ctx context.Context,
	hk, id string,
) error {
	if s.ArchiveSnapshotsFunc != nil {
		return s.ArchiveSnapshotsFunc(ctx, hk, id)
	}

	if s.SnapshotWriter != nil {
		return s.SnapshotWriter.ArchiveSnapshots(ctx, hk, id)
	}

	return nil
}
