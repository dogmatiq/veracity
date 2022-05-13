package aggregate

import (
	"context"
	"fmt"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/linger"
)

// DefaultSnapshotWriteTimeout is the default amount of time to allow for
// writing a snapshot during the load process.
const DefaultSnapshotWriteTimeout = 100 * time.Millisecond

// Loader loads aggregate roots from their snapshots and historical events.
type Loader struct {
	// EventReader is used to read historical events.
	EventReader EventReader

	// SnapshotReader is used to read snapshots of aggregate roots from
	// persistent storage.
	//
	// It may be nil, in which case aggregate roots are always loaded by reading
	// all of their historical events (which may take significant time).
	SnapshotReader SnapshotReader

	// SnapshotWriter is used to persist snapshots of an aggregate root.
	//
	// New snapshots are persisted if a load operation fails before all
	// historical events can be read. Persisting a new snapshot reduces the
	// number of historical events next time the root is loaded.
	//
	// It may be nil, in which case no snapshots are persisted.
	SnapshotWriter SnapshotWriter

	// SnapshotWriteTimeout is the maximum duration to allow for a snapshot to
	// be written.
	//
	// If it is non-positive, DefaultSnapshotWriteTimeout is used instead.
	SnapshotWriteTimeout time.Duration

	// Logger is the target for messages about the loading process.
	Logger logging.Logger
}

// Load populates an aggregate root from a snapshot and/or its historical
// events.
//
// root must be a zero-value aggregate root obtained by calling
// AggregateMessageHandler.New().
//
// The root is first populated with state from the most recent snapshot. Then
// any historical events that occurred after the snapshot are applied to the
// root.
//
// If some (but not all) historical events are able to be applied to the root a
// new snapshot is persisted, reducing the number of historical events that need
// to be applied the next time this instance is loaded. This can help combat
// read failures that occur as a result of timeouts due to a large number of
// historical events.
//
// It returns nextOffset, which is the offset of the next event that will be
// applied to the root.
func (l *Loader) Load(
	ctx context.Context,
	handler configkit.Identity,
	instanceID string,
	root dogma.AggregateRoot,
) (nextOffset uint64, _ error) {
	// First read the event bounds that we are interested in. This helps us
	// optimize the reads we perform by ignoring any snapshots and events that
	// are no longer relevant.
	firstOffset, nextOffset, err := l.EventReader.ReadBounds(
		ctx,
		handler.Key,
		instanceID,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read event offset bounds: %w",
			handler.Name,
			instanceID,
			err,
		)
	}

	// If the nextOffset is zero this instance has no events at all.
	if nextOffset == 0 {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] has no historical events",
			handler.Name,
			instanceID,
		)

		return 0, nil
	}

	// If the firstOffset and the nextOffset are the same then the instance has
	// historical events, but it has been destroyed since those events were
	// recorded and hence the current state is the zero-value.
	if firstOffset == nextOffset {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] has no historical events since being destroyed at offset %s",
			handler.Name,
			instanceID,
			firstOffset-1,
		)

		return nextOffset, nil
	}

	// Attempt to read a snapshot so that we don't have to read the entire set
	// of historical events.
	snapshotOffset, hasSnapshot, err := l.readSnapshot(
		ctx,
		handler,
		instanceID,
		firstOffset, // ignore any snapshots from before the instance was last destroyed
		root,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read snapshot: %w",
			handler.Name,
			instanceID,
			err,
		)
	}

	if hasSnapshot {
		// We found a snapshot, so the first event we want to read is the one
		// after that snapshot was taken.
		firstOffset = snapshotOffset + 1

		// If the firstOffset after the snapshot is the same as the nextOffset
		// then the snapshot is up-to-date and no historical events need to be
		// read.
		if firstOffset == nextOffset {
			logging.Log(
				l.Logger,
				"aggregate root %s[%s] loaded from up-to-date snapshot taken at offset %s",
				handler.Name,
				instanceID,
				snapshotOffset,
			)

			return nextOffset, nil
		}
	}

	// Finally, we read and apply the historical events that occurred after the
	// snapshot (or since the instance was last destroyed if there is no
	// snapshot available).
	nextOffset, err = l.readEvents(
		ctx,
		handler,
		instanceID,
		firstOffset,
		root,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read events: %w",
			handler.Name,
			instanceID,
			err,
		)
	}

	if hasSnapshot {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] loaded from outdated snapshot taken at offset %d and %d subsequent event(s)",
			handler.Name,
			instanceID,
			snapshotOffset,
			nextOffset-snapshotOffset-1,
		)
	} else {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] loaded from %d event(s)",
			handler.Name,
			instanceID,
			nextOffset-firstOffset,
		)
	}

	return nextOffset, nil
}

// readSnapshot attempts to read a snapshot of an aggregate instance.
//
// ok is true if a snapshot is available, in which case snapshotOffset is the
// offset at which the snapshot was taken.
//
// A failure to load a snapshot is not treated as an error. err is non-nil only
// if ctx is canceled.
func (l *Loader) readSnapshot(
	ctx context.Context,
	handler configkit.Identity,
	instanceID string,
	firstOffset uint64,
	root dogma.AggregateRoot,
) (snapshotOffset uint64, ok bool, err error) {
	// Snapshots are entirely optional, bail if no reader is configured.
	if l.SnapshotReader == nil {
		return 0, false, nil
	}

	snapshotOffset, ok, err = l.SnapshotReader.ReadSnapshot(
		ctx,
		handler.Key,
		instanceID,
		firstOffset,
		root,
	)
	if err != nil {
		// If the error was due to a context cancelation/timeout we bail with
		// the context error.
		if err == ctx.Err() {
			return 0, false, err
		}

		// Otherwise it's not a fatal error, so we log about it but don't return
		// an error. We may still be able to load the aggregate root from its
		// entire event history.
		logging.Log(
			l.Logger,
			"snapshot of aggregate root %s[%s] cannot be read: %w",
			handler.Name,
			instanceID,
			err,
		)

		return 0, false, nil
	}

	return snapshotOffset, ok, nil
}

// writeSnapshot attempts to write a snapshot of an aggregate instance.
//
// snapshotOffset is the offset of the last event to be applied to root, which
// is not necessarily the instance's most recent event.
func (l *Loader) writeSnapshot(
	handler configkit.Identity,
	instanceID string,
	snapshotOffset uint64,
	root dogma.AggregateRoot,
) {
	// Snapshots are entirely optional, bail if no writer is configured.
	if l.SnapshotWriter == nil {
		return
	}

	// Setup a timeout independent of the context passed to Load(). This is
	// important as it's what allows the snapshot to be written even if the
	// initial load failed due to the context deadline being exceeded.
	ctx, cancel := linger.ContextWithTimeout(
		context.Background(),
		l.SnapshotWriteTimeout,
		DefaultSnapshotWriteTimeout,
	)
	defer cancel()

	if err := l.SnapshotWriter.WriteSnapshot(
		ctx,
		handler.Key,
		instanceID,
		snapshotOffset,
		root,
	); err != nil {
		logging.Log(
			l.Logger,
			"outdated snapshot of aggregate root %s[%s] cannot be be written at offset %d: %w",
			handler.Name,
			instanceID,
			snapshotOffset,
			err,
		)

		return
	}

	logging.Log(
		l.Logger,
		"outdated snapshot of aggregate root %s[%s] written at offset %d",
		handler.Name,
		instanceID,
		snapshotOffset,
	)
}

// readEvents loads historical events and applies them to root.
//
// startOffset specifies the offset of the first event to read.
//
// If reading events fails before all historical events can be read, a new
// snapshot is taken in an effort to reduce the number of historical events that
// need to be read on subsequent attempts to load this instance.
//
// It returns the offset of the next event to be recorded by this instance. Note
// this may be greater than the nextOffset loaded by the initial bounds check.
func (l *Loader) readEvents(
	ctx context.Context,
	handler configkit.Identity,
	instanceID string,
	startOffset uint64,
	root dogma.AggregateRoot,
) (nextOffset uint64, err error) {
	nextOffset = startOffset

	for {
		events, err := l.EventReader.ReadEvents(
			ctx,
			handler.Key,
			instanceID,
			nextOffset,
		)
		if err != nil {
			if nextOffset > startOffset {
				// We managed to read _some_ events before this error occurred,
				// so we write a new snapshot so that we can "pick up where we
				// left off" next time we attempt to load this instance.
				l.writeSnapshot(
					handler,
					instanceID,
					nextOffset-1, // snapshot offset
					root,
				)
			}

			return nextOffset, err
		}

		if len(events) == 0 {
			return nextOffset, nil
		}

		for _, ev := range events {
			root.ApplyEvent(ev)
			startOffset++
		}
	}
}
