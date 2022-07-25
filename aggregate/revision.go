package aggregate

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
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

// Bounds describes the revisions that are the bounds of the relevant historical
// revisions for a specific aggregate instance.
type Bounds struct {
	// Begin is the (inclusive) first revision of the aggregate instance that is
	// relevant to its state.
	//
	// When loading the instance, only those events from revisions in the
	// half-open range [begin, end) should be applied to the aggregate
	// root.
	Begin uint64

	// End is the (exclusive) end revision of the aggregate instance.
	End uint64

	// Committed is the (exclusive) revision of latest comitted revision.
	//
	// That is, only the revisions in the half-open range [0, committed)
	// have been committed. Committed may be less than Begin.
	Committed uint64
}

// RevisionReader is an interface for reading historical revisions recorded by
// aggregate instances.
type RevisionReader interface {
	// ReadBounds returns the revisions that are the bounds of the relevant
	// historical events for a specific aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	ReadBounds(
		ctx context.Context,
		hk, id string,
	) (Bounds, error)

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
