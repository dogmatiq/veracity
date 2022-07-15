package aggregate

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// EventReader is an interface for reading historical events recorded by
// aggregate instances.
type EventReader interface {
	// ReadBounds returns the revisions that are the bounds of the relevant
	// historical events for a specific aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// When loading the instance, only those events from revisions in the
	// half-open range [begin, end) should be applied to the aggregate root.
	ReadBounds(
		ctx context.Context,
		hk, id string,
	) (begin, end uint64, _ error)

	// ReadEvents loads some historical events for a specific aggregate
	// instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// It returns an implementation-defined number of events sourced from
	// revisions in the half-open range [begin, end).
	//
	// When begin == end there are no more historical events to read. Otherwise,
	// call ReadEvents() again with begin = end to continue reading events.
	//
	// The behavior is undefined if begin is lower than the begin revision
	// returned by ReadBounds(). Implementations should return an error in this
	// case.
	ReadEvents(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (events []*envelopespec.Envelope, end uint64, _ error)
}

// EventWriter is an interface for persisting the events recorded by aggregate
// instances.
type EventWriter interface {
	// WriteEvents writes events that were recorded by an aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// begin sets the first revision for the instance such that in the future
	// only events from revisions in the half-open range [begin, end + 1) are
	// applied when loading the aggregate root. The behavior is undefined if
	// begin is larger than end + 1.
	//
	// Events from revisions prior to begin are still made available to external
	// event consumers, but will no longer be needed for loading aggregate roots
	// and may be archived.
	//
	// end must be the current end revision, that is, the revision after the
	// most recent revision of the instance. Otherwise, an "optimistic
	// concurrency control" error occurs and no changes are persisted. The
	// behavior is undefined if end is greater than the actual end revision.
	//
	// The events slice may be empty, which allows modifying the begin revision
	// without recording any new events.
	WriteEvents(
		ctx context.Context,
		hk, id string,
		begin, end uint64,
		events []*envelopespec.Envelope,
	) error
}
