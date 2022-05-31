package aggregate

import (
	"context"
	"time"

	"github.com/dogmatiq/dogma"
)

const (
	// DefaultSnapshotInterval is the default number of events that can be
	// recorded by an aggregate instance before it is considered neccessary to
	// take a new snapshot.
	DefaultSnapshotInterval = 1000

	// DefaultSnapshotWriteTimeout is the default amount of time to allow for
	// writing a snapshot.
	DefaultSnapshotWriteTimeout = 250 * time.Millisecond
)

// SnapshotReader is an interface for reading snapshots of aggregate roots from
// persistent storage.
type SnapshotReader interface {
	// ReadSnapshot updates the contents of r to match the most recent snapshot
	// that was taken at or after minOffset.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// If ok is false, no compatible snapshot was found at or after minOffset;
	// root is guaranteed not to have been modified. Otherwise, snapshotOffset
	// is the offset of the most recent event applied to the root when the
	// snapshot was taken.
	//
	// A snapshot is considered compatible if it can assigned to the underlying
	// type of r.
	ReadSnapshot(
		ctx context.Context,
		hk, id string,
		r dogma.AggregateRoot,
		minOffset uint64,
	) (snapshotOffset uint64, ok bool, _ error)
}

// SnapshotWriter is an interface for writing snapshots of aggregate roots to
// persistent storage.
type SnapshotWriter interface {
	// WriteSnapshot saves a snapshot of a specific aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// snapshotOffset is the offset of the most recent event that has been
	// applied to the r.
	WriteSnapshot(
		ctx context.Context,
		hk, id string,
		r dogma.AggregateRoot,
		snapshotOffset uint64,
	) error

	// ArchiveSnapshots archives any existing snapshots of a specific instance.
	//
	// The precise meaning of "archive" is implementation-defined. It is typical
	// to hard-delete the snapshots as they no longer serve a purpose and will
	// not be required in the future.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	ArchiveSnapshots(
		ctx context.Context,
		hk, id string,
	) error
}
