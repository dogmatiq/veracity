package aggregate

import (
	"context"
	"fmt"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/marshalkit"
	"go.uber.org/zap"
)

// Revision is a single revision of an aggregate instance.
type Revision struct {
	// Begin is the (inclusive) first revision of the aggregate instance that is
	// relevant to its state at the time this revision was prepared. It may have
	// been modified by this revision.
	Begin uint64

	// End is the (exclusive) end revision at the time this revision was
	// prepared. Therefore, it is the index of this revision within the
	// aggregate's history.
	End uint64

	// Events are the events recorded within this revision.
	Events []*envelopespec.Envelope
}

// RevisionReader is an interface for reading historical revisions recorded by
// aggregate instances.
type RevisionReader interface {
	// ReadBounds returns the revisions that are the bounds of the relevant
	// historical events for a specific aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// commited is the revision after the most recent committed revision,
	// whereas end is the revision after the most recent revision, regardless of
	// whether it is committed.
	//
	// When loading the instance, only those events from revisions in the
	// half-open range [begin, end) should be applied to the aggregate
	// root.
	ReadBounds(
		ctx context.Context,
		hk, id string,
	) (begin, committed, end uint64, _ error)

	// ReadRevisions loads some historical revisions for a specific aggregate
	// instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// It returns an implementation-defined number of revisions starting with
	// the begin revision.
	//
	// When the revisions slice is empty there are no more revisions to read.
	// Otherwise, begin should be incremented by len(revisions) and ReadEvents()
	// called again to continue reading revisions.
	//
	// The behavior is undefined if begin is lower than the begin revision
	// returned by ReadBounds(). Implementations should return an error in this
	// case.
	ReadRevisions(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (revisions []Revision, _ error)
}

// RevisionWriter is an interface for persisting new revisions of aggregate
// instances.
type RevisionWriter interface {
	// PrepareRevision prepares a new revision of an aggregate instance to be
	// committed.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// rev.Begin sets the first revision for the instance. In the future only
	// events from revisions in the half-open range [rev.Begin, rev.End + 1) are
	// applied when loading the aggregate root. The behavior is undefined if
	// rev.Begin is larger than rev.End + 1.
	//
	// Events from revisions prior to rev.Begin are still made available to
	// external event consumers, but will no longer be needed for loading
	// aggregate roots and may be archived.
	//
	// rev.End must be the current end revision, that is, the revision after the
	// most recent revision of the instance. Otherwise, an "optimistic
	// concurrency control" error occurs and no changes are persisted. The
	// behavior is undefined if rev.End is greater than the actual end revision.
	PrepareRevision(
		ctx context.Context,
		hk, id string,
		rev Revision,
	) error

	// CommitRevision commits a prepared revision.
	//
	// Only once a revision is committed can it be considered part of the
	// aggregate's history.
	//
	// rev must exactly match the revision that was prepared. Care must be taken
	// as his may or may not be enforced by the implementation.
	//
	// It returns an error if the revision does not exist or has already been
	// committed.
	//
	// It returns an error if there are uncommitted revisions before the given
	// revision.
	CommitRevision(
		ctx context.Context,
		hk, id string,
		rev Revision,
	) error
}

// RevisionLoader loads aggregate roots using a RevisionReader.
type RevisionLoader struct {
	// RevisionReader is used to read revisions from persistent storage.
	RevisionReader RevisionReader

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
	// revisions can be read. Persisting a new snapshot reduces the number of
	// historical events that need to be applied next time the root is loaded.
	//
	// It may be nil, in which case no snapshots are persisted.
	SnapshotWriter SnapshotWriter

	// SnapshotWriteTimeout is the maximum duration to allow for a snapshot to
	// be written to persistent storage.
	//
	// If it is non-positive, DefaultLoaderSnapshotWriteTimeout is used instead.
	SnapshotWriteTimeout time.Duration

	// Logger is the target for messages about the loading process.
	Logger *zap.Logger
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
// If an error occurs before events from all historical revisions can be applied
// to r, a new snapshot is taken as of the most recently applied revision, then
// the error is returned. This can help prevent timeouts that occur while
// reading a large number of historical events by making a more up-to-date
// snapshot available next time the instance is loaded.
//
// Load never returns an error as a result of reading or writing snapshots; it
// will always fall-back to using the historical events.
//
// The half-open range [begin, end) is the range of unarchived revisions that
// wre used to populate r.
//
// snapshotAge is the number of revisions that have been made since the last
// snapshot was taken.
func (l *RevisionLoader) Load(
	ctx context.Context,
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
) (begin, end, snapshotAge uint64, _ error) {
	// First read the bounds that we are interested in. This helps us optimize
	// the reads we perform by ignoring any snapshots and revisions that are no
	// longer relevant.
	begin, _, end, err := l.RevisionReader.ReadBounds(
		ctx,
		h.Key,
		id,
	)
	if err != nil {
		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read revision bounds: %w",
			h.Name,
			id,
			err,
		)
	}

	// If the end is zero this instance has no revisions at all.
	if end == 0 {
		l.Logger.Debug(
			"aggregate root has no historical revisions",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
		)

		return 0, 0, 0, nil
	}

	// If begin and end are the same then the instance was destroyed at the same
	// time the most recent revision was made, and therefore no events need to
	// be applied.
	if begin == end {
		l.Logger.Debug(
			"aggregate root has no historical revisions since it was last destroyed",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
			zap.Uint64("destruction_revision", begin-1),
		)

		return begin, end, 0, nil
	}

	// Attempt to read a snapshot that is newer than begin so that we don't have
	// to read the entire set of historical revisions.
	snapshotRev, hasSnapshot, err := l.readSnapshot(
		ctx,
		h,
		id,
		r,
		begin, // ignore any snapshots that were taken before the first relevant revision
	)
	if err != nil {
		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read snapshot: %w",
			h.Name,
			id,
			err,
		)
	}

	readFrom := begin

	if hasSnapshot {
		// We found a snapshot, so the first revision we want to read is the one
		// after that snapshot was taken.
		readFrom = snapshotRev + 1

		// If the revision after the snapshot is the same as the end then no
		// revisions have been made since the snapshot was taken, and therefore
		// there's nothing more to load.
		if readFrom == end {
			l.Logger.Debug(
				"aggregate root loaded from up-to-date snapshot",
				zap.String("handler_name", h.Name),
				zap.String("handler_key", h.Key),
				zap.String("instance_id", id),
				zap.Uint64("snapshot_revision", snapshotRev),
			)

			return begin, end, 0, nil
		}
	}

	// Finally, we read revisions and apply the historical events that occurred
	// after the snapshot (or since the instance was last destroyed if there is
	// no snapshot available).
	end, err = l.readRevisions(
		ctx,
		h,
		id,
		readFrom,
		r,
	)
	if err != nil {
		if end > readFrom {
			// We managed to read _some_ revisions before this error occurred,
			// so we write a new snapshot so that we can "pick up where we left
			// off" next time we attempt to load this instance.
			l.writeSnapshot(
				h,
				id,
				r,
				end-1, // snapshot revision
			)
		}

		return 0, 0, 0, fmt.Errorf(
			"aggregate root %s[%s] cannot be loaded: unable to read revisions: %w",
			h.Name,
			id,
			err,
		)
	}

	snapshotAge = end - readFrom

	// Log about how the aggregate root was loaded.
	if hasSnapshot {
		l.Logger.Debug(
			"aggregate root loaded from out-of-date snapshot",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
			zap.Uint64("snapshot_revision", snapshotRev),
			zap.Uint64("snapshot_age", snapshotAge),
		)
	} else {
		l.Logger.Debug(
			"aggregate root loaded without using a snapshot",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
			zap.Uint64("revisions", snapshotAge),
		)
	}

	return begin, end, snapshotAge, nil
}

// readSnapshot updates the contents of r to match the most recent snapshot that
// was taken at or after minRev.
//
// ok is true if such a snapshot is available, in which case rev is the revision
// at which the snapshot was taken. If ok is false, r is guaranteed not to have
// been modified.
//
// A failure to load a snapshot is not treated as an error; err is non-nil only
// if ctx is canceled.
func (l *RevisionLoader) readSnapshot(
	ctx context.Context,
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
	minRev uint64,
) (rev uint64, ok bool, err error) {
	// Snapshots are entirely optional, bail if no reader is configured.
	if l.SnapshotReader == nil {
		return 0, false, nil
	}

	rev, ok, err = l.SnapshotReader.ReadSnapshot(
		ctx,
		h.Key,
		id,
		r,
		minRev,
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
		l.Logger.Warn(
			"cannot read snapshot of aggregate root",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
			zap.Error(err),
		)

		return 0, false, nil
	}

	return rev, ok, nil
}

// writeSnapshot writes a snapshot of an aggregate root.
//
// rev is the revision of the aggregate instance as represented by r, which is
// not necessarily the instance's most recent revision.
func (l *RevisionLoader) writeSnapshot(
	h configkit.Identity,
	id string,
	r dogma.AggregateRoot,
	rev uint64,
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
		rev,
	); err != nil {
		l.Logger.Warn(
			"cannot write (possibly outdated) snapshot of aggregate root",
			zap.String("handler_name", h.Name),
			zap.String("handler_key", h.Key),
			zap.String("instance_id", id),
			zap.Uint64("revision", rev),
			zap.Error(err),
		)
		return
	}

	l.Logger.Debug(
		"wrote (possibly outdated) snapshot of aggregate root",
		zap.String("handler_name", h.Name),
		zap.String("handler_key", h.Key),
		zap.String("instance_id", id),
		zap.Uint64("revision", rev),
	)
}

// readRevisions loads revisions and applies historical events to r.
//
// It reads revisions in the half-open range [begin, end). end is valid even
// even if err is non-nil.
func (l *RevisionLoader) readRevisions(
	ctx context.Context,
	h configkit.Identity,
	id string,
	begin uint64,
	r dogma.AggregateRoot,
) (end uint64, err error) {
	var events []dogma.Message
	end = begin

	for {
		revisions, err := l.RevisionReader.ReadRevisions(
			ctx,
			h.Key,
			id,
			begin,
		)
		if err != nil {
			return end, err
		}

		if len(revisions) == 0 {
			return end, nil
		}

		for _, rev := range revisions {
			events = events[:0]

			// Unmarshal all of the envelopes before applying them to guarantee
			// that we can apply all events from the revision.
			for _, env := range rev.Events {
				ev, err := marshalkit.UnmarshalMessageFromEnvelope(l.Marshaler, env)
				if err != nil {
					return end, err
				}

				events = append(events, ev)
			}

			for _, ev := range events {
				r.ApplyEvent(ev)
			}

			end++
		}

		begin = end
	}
}
