package aggregate

import (
	"context"
	"fmt"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/marshalkit"
)

const (
	// DefaultLoaderSnapshotWriteTimeout is the default amount of time to allow
	// for writing a snapshot after a failure to read events.
	DefaultLoaderSnapshotWriteTimeout = 250 * time.Millisecond
)

// Loader loads aggregate roots from persistent storage.
type Loader struct {
	// EventReader is used to read historical events from persistent storage.
	EventReader EventReader

	// Marshaler is used to unmarshal historical events.
	Marshaler marshalkit.ValueMarshaler

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
	// number of historical events that need to be read next time the root is
	// loaded.
	//
	// It may be nil, in which case no snapshots are persisted.
	SnapshotWriter SnapshotWriter

	// SnapshotWriteTimeout is the maximum duration to allow for a snapshot to
	// be written to persistent storage.
	//
	// If it is non-positive, DefaultLoaderSnapshotWriteTimeout is used instead.
	SnapshotWriteTimeout time.Duration

	// Logger is the target for messages about the loading process.
	Logger logging.Logger
}

// Load populates an aggregate root with data loaded from persistent storage.
//
// h is the identity of the handler that implements the aggregate. id is the
// instance of the aggregate to load.
//
// r must be an aggregate root that represents the "initial state" of the
// aggregate. It is typically a zero-value obtained by calling the
// AggregateMessageHandler.New() method on the handler identified by h.
//
// The contents of r is first updated to match the most recent snapshot of the
// instance. Then, any historical events that were recorded by the instance
// after that snapshot was taken are applied by calling r.ApplyEvent(). If no
// snapshot is available, r is populated by applying all of its historical
// events, which may take significant time.
//
// If an error occurs before all historical events can be applied to r, a new
// snapshot is taken as of the most recently applied event, then the error is
// returned. This can help prevent timeouts that occur while reading a large
// number of historical events by making a more up-to-date snapshot available
// next time the instance is loaded.
//
// Load never returns an error as a result of reading or writing snapshots; it
// will always fall-back to using the historical events.
//
// It returns firstOffset, which is the offset of the first non-archived event;
// and nextOffset, which is the offset that will be occupied by the next event
// to be recorded by this instance.
//
// snapshotAge is the number of events that have been recorded since the last
// snapshot was taken.
func (l *Loader) Load(
	ctx context.Context,
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
) (firstOffset, nextOffset, snapshotAge uint64, _ error) {
	// First read the event bounds that we are interested in. This helps us
	// optimize the reads we perform by ignoring any snapshots and events that
	// are no longer relevant.
	firstOffset, nextOffset, err := l.EventReader.ReadBounds(
		ctx,
		h.Key,
		id,
	)
	if err != nil {
		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read event offset bounds: %w",
			h.Name,
			id,
			err,
		)
	}

	// If the nextOffset is zero this instance has no events at all.
	if nextOffset == 0 {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] has no historical events",
			h.Name,
			id,
		)

		return 0, 0, 0, nil
	}

	// If the firstOffset and the nextOffset are the same then the instance was
	// destroyed at the same time the most recent event was recorded and
	// therefore no events need to be applied.
	if firstOffset == nextOffset {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] has no historical events since being destroyed at offset %d",
			h.Name,
			id,
			firstOffset-1,
		)

		return firstOffset, nextOffset, 0, nil
	}

	// Attempt to read a snapshot that is newer than firstOffset so that we
	// don't have to read the entire set of historical events.
	snapshotOffset, hasSnapshot, err := l.readSnapshot(
		ctx,
		h,
		id,
		r,
		firstOffset, // ignore any snapshots that were taken before the first relevant event
	)
	if err != nil {
		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read snapshot: %w",
			h.Name,
			id,
			err,
		)
	}

	// Start reading events at the first non-archived offset.
	readOffset := firstOffset

	if hasSnapshot {
		// We found a snapshot, so the first event we want to read is the one
		// after that snapshot was taken.
		readOffset = snapshotOffset + 1

		// If the firstReadOffset after the snapshot is the same as the
		// nextOffset then no events have occurred since the snapshot was taken,
		// and therefore there's nothing more to load.
		if readOffset == nextOffset {
			logging.Log(
				l.Logger,
				"aggregate root %s[%s] loaded from up-to-date snapshot taken at offset %d",
				h.Name,
				id,
				snapshotOffset,
			)

			return firstOffset, nextOffset, 0, nil
		}
	}

	// Finally, we read and apply the historical events that occurred after the
	// snapshot (or since the instance was last destroyed if there is no
	// snapshot available).
	nextOffset, err = l.readEvents(
		ctx,
		h,
		id,
		r,
		readOffset,
	)
	if err != nil {
		if nextOffset > readOffset {
			// We managed to read _some_ events before this error occurred,
			// so we write a new snapshot so that we can "pick up where we
			// left off" next time we attempt to load this instance.
			l.writeSnapshot(
				h,
				id,
				r,
				nextOffset-1, // snapshot offset
			)
		}

		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read events: %w",
			h.Name,
			id,
			err,
		)
	}

	snapshotAge = nextOffset - readOffset

	// Log about how the aggregate root was loaded.
	if hasSnapshot {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] loaded from outdated snapshot taken at offset %d and %d subsequent event(s)",
			h.Name,
			id,
			snapshotOffset,
			snapshotAge,
		)
	} else {
		logging.Log(
			l.Logger,
			"aggregate root %s[%s] loaded from %d event(s)",
			h.Name,
			id,
			snapshotAge,
		)
	}

	return firstOffset, nextOffset, snapshotAge, nil
}

// readSnapshot updates the contents of r to match the most recent snapshot that
// was taken at or after minOffset.
//
// ok is true if such a snapshot is available, in which case snapshotOffset is
// the offset at which the snapshot was taken. If ok is false, r is guaranteed
// not to have been modified.
//
// A failure to load a snapshot is not treated as an error; err is non-nil only
// if ctx is canceled.
func (l *Loader) readSnapshot(
	ctx context.Context,
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
	minOffset uint64,
) (snapshotOffset uint64, ok bool, err error) {
	// Snapshots are entirely optional, bail if no reader is configured.
	if l.SnapshotReader == nil {
		return 0, false, nil
	}

	snapshotOffset, ok, err = l.SnapshotReader.ReadSnapshot(
		ctx,
		h.Key,
		id,
		r,
		minOffset,
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
			h.Name,
			id,
			err,
		)

		return 0, false, nil
	}

	return snapshotOffset, ok, nil
}

// writeSnapshot writes a snapshot of an aggregate root.
//
// snapshotOffset is the offset of the last event to be applied to r, which
// is not necessarily the instance's most recent event.
func (l *Loader) writeSnapshot(
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
	snapshotOffset uint64,
) {
	// Snapshots are entirely optional, bail if no writer is configured.
	if l.SnapshotWriter == nil {
		return
	}

	// Setup a timeout that is independent of the context passed to Load().
	//
	// This is important as it allows the snapshot to be written even if the
	// initial load failed due to the context deadline being exceeded.
	ctx, cancel := linger.ContextWithTimeout(
		context.Background(),
		l.SnapshotWriteTimeout,
		DefaultLoaderSnapshotWriteTimeout,
	)
	defer cancel()

	if err := l.SnapshotWriter.WriteSnapshot(
		ctx,
		h.Key,
		id,
		r,
		snapshotOffset,
	); err != nil {
		logging.Log(
			l.Logger,
			"snapshot of aggregate root %s[%s] cannot be be written at offset %d (possibly outdated): %w",
			h.Name,
			id,
			snapshotOffset,
			err,
		)

		return
	}

	logging.Log(
		l.Logger,
		"snapshot of aggregate root %s[%s] written at offset %d (possibly outdated)",
		h.Name,
		id,
		snapshotOffset,
	)
}

// readEvents loads historical events and applies them to r.
//
// firstOffset is the offset of the first event to read.
//
// It returns the offset of the next event to be recorded by this instance. Note
// this may be greater than the nextOffset loaded by the initial bounds check.
// The next offset is returned even if err is non-nil.
func (l *Loader) readEvents(
	ctx context.Context,
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
	firstOffset uint64,
) (nextOffset uint64, err error) {
	nextOffset = firstOffset

	for {
		envelopes, more, err := l.EventReader.ReadEvents(
			ctx,
			h.Key,
			id,
			nextOffset,
		)
		if err != nil {
			return nextOffset, err
		}

		for _, env := range envelopes {
			ev, err := marshalkit.UnmarshalMessageFromEnvelope(l.Marshaler, env)
			if err != nil {
				return nextOffset, err
			}

			r.ApplyEvent(ev)
			nextOffset++
		}

		if !more {
			return nextOffset, nil
		}
	}
}
