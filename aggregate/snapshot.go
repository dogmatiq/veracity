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
	// ReadSnapshot loads the most recent snapshot of a specific aggregate
	// instance.
	//
	// It populates the given aggregate root with the snapshot data.
	//
	// It returns the offset of the last event applied to the root when the
	// snapshot was taken.
	//
	// If ok is false, no snapshot was found at or after minOffset; root is
	// guaranteed not to have been modified.
	ReadSnapshot(
		ctx context.Context,
		handlerKey, instanceID string,
		minOffset uint64,
		root dogma.AggregateRoot,
	) (_ uint64, ok bool, _ error)
}

// SnapshotWriter is an interface for writing snapshots of aggregate roots to
// persistent storage.
type SnapshotWriter interface {
	// WriteSnapshot saves a snapshot of a specific aggregate instance.
	//
	// offset is the offset of the most recent event that has been applied to
	// the root.
	WriteSnapshot(
		ctx context.Context,
		handlerKey, instanceID string,
		offset uint64,
		root dogma.AggregateRoot,
	) error

	// DeleteSnapshots deletes all snapshots of a specific aggregate instance.
	DeleteSnapshots(
		ctx context.Context,
		handlerKey, instanceID string,
	) error
}
