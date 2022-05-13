package aggregate

import (
	"context"
	"time"

	"github.com/dogmatiq/dogma"
)

// DefaultSnapshotWriteTimeout is the default amount of time to allow for
// writing a snapshot.
const DefaultSnapshotWriteTimeout = 100 * time.Millisecond

// SnapshotReader is an interface for reading snapshots of aggregate roots from
// persistent storage.
type SnapshotReader interface {
	// ReadSnapshot updates the contents of r to match the most recent snapshot
	// that was taken at or after minOffset.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// If ok is false, no snapshot was found at or after minOffset; root is
	// guaranteed not to have been modified. Otherwise, snapshotOffset is the
	// offset of the most recent event applied to the root when the snapshot was
	// taken.
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

	// DeprecateSnapshots indicates to the implementation that any existing
	// snapshots of a specific instance are no longer required.
	//
	// Implementations would typically delete the snapshots, but may choose to
	// archive them or even do nothing.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	DeprecateSnapshots(
		ctx context.Context,
		hk, id string,
	) error
}
