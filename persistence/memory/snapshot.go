package memory

import (
	"context"
	"reflect"
	"sync"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
)

// AggregateSnapshotStore stores snapshots of aggregate roots in memory.
//
// It implements aggregate.SnapshotReader and aggregate.SnapshotWriter.
type AggregateSnapshotStore struct {
	Marshaler marshalkit.ValueMarshaler

	m         sync.RWMutex
	snapshots map[instanceKey]snapshot
}

type snapshot struct {
	Packet marshalkit.Packet
	Offset uint64
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
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	sn, ok := s.snapshots[k]
	if !ok {
		return 0, false, nil
	}

	if sn.Offset < minOffset {
		return 0, false, nil
	}

	v, err := s.Marshaler.Unmarshal(sn.Packet)
	if err != nil {
		return 0, false, err
	}

	src := reflect.ValueOf(v).Elem()
	dst := reflect.ValueOf(r).Elem()

	if !src.Type().AssignableTo(dst.Type()) {
		return 0, false, nil
	}

	dst.Set(src)

	return sn.Offset, true, nil
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
	p, err := s.Marshaler.Marshal(r)
	if err != nil {
		return err
	}

	s.m.Lock()
	defer s.m.Unlock()

	if s.snapshots == nil {
		s.snapshots = map[instanceKey]snapshot{}
	}

	k := instanceKey{hk, id}
	s.snapshots[k] = snapshot{p, snapshotOffset}

	return nil
}

// ArchiveSnapshots archives any existing snapshots of a specific aggregate
// instance.
//
// The precise meaning of "archive" is implementation-defined. It is typical to
// hard-delete the snapshots as they no longer serve a purpose and will not be
// required in the future.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
func (s *AggregateSnapshotStore) ArchiveSnapshots(
	ctx context.Context,
	hk, id string,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	delete(s.snapshots, k)

	return nil
}
