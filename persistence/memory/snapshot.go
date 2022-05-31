package memory

import (
	"context"
	"reflect"

	"github.com/dogmatiq/dogma"
)

// AggregateSnapshotStore stores snapshots of aggregate roots in memory.
//
// It implements aggregate.SnapshotReader and aggregate.SnapshotWriter.
type AggregateSnapshotStore struct {
	root   reflect.Value
	offset uint64
}

// ReadSnapshot updates the contents of r to match the most recent snapshot that
// was taken at or after minOffset.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// If ok is false, no compatible snapshot was found at or after minOffset; root
// is guaranteed not to have been modified. Otherwise, snapshotOffset is the
// offset of the most recent event applied to the root when the snapshot was
// taken.
//
// A snapshot is considered compatible if it can assigned to the underlying type
// of r.
func (s *AggregateSnapshotStore) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minOffset uint64,
) (snapshotOffset uint64, ok bool, _ error) {
	if !s.root.IsValid() {
		return 0, false, nil
	}

	if s.offset < minOffset {
		return 0, false, nil
	}

	src := s.root.Elem()
	dst := reflect.ValueOf(r).Elem()

	if !src.Type().AssignableTo(dst.Type()) {
		return 0, false, nil
	}

	dst.Set(src)

	return s.offset, true, nil
}

// WriteSnapshot saves a snapshot of a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the
// aggregate instance ID.
//
// snapshotOffset is the offset of the most recent event that has been
// applied to the r.
func (s *AggregateSnapshotStore) WriteSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	snapshotOffset uint64,
) error {
	s.root = reflect.ValueOf(r)
	s.offset = snapshotOffset

	return nil
}

// ArchiveSnapshots archives any existing snapshots of a specific instance
// are no longer required.
//
// The precise meaning of "archive" is implementation-defined. It is typical
// to hard-delete the snapshots as they no longer serve a purpose and will
// not be required in the future.
//
// hk is the identity key of the aggregate message handler. id is the
// aggregate instance ID.
func (s *AggregateSnapshotStore) ArchiveSnapshots(
	ctx context.Context,
	hk, id string,
) error {
	panic("not implemented")
}
