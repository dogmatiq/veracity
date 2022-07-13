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
	// begin is the earliest (inclusive) revision that is considered relevant to
	// the current state of the instance. It begins as zero, but is advanced as
	// events are archived.
	//
	// end is latest (exclusive) revision of the instance. Because it is
	// exclusive it is equivalent to the next revision of this instance.
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
	// begin is the earliest (inclusive) revision to include in the result.
	//
	// events is the set of events starting at the begin revision.
	//
	// If end > begin, there are more events to be loaded, and ReadEvents()
	// should be called again with begin set to end.
	//
	// If end == begin there are no subsequent historical events to be loaded.
	//
	// The number of revisions (and hence events) returned by a single call to
	// ReadEvents() is implementation defined.
	//
	// It may return an error if begin refers to an archived revision.
	ReadEvents(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (events []*envelopespec.Envelope, end uint64, _ error)
}

// EventWriter is an interface for recording the events produced by aggregate
// instances.
type EventWriter interface {
	// WriteEvents writes events that were recorded by an aggregate instance.
	//
	// hk is the identity key of the aggregate message handler. id is the
	// aggregate instance ID.
	//
	// end must be the revision after the most recent revision of the instance;
	// otherwise an "optimistic concurrency control" error occurs and no changes
	// are persisted.
	//
	// If archive is true, all historical events, including those written by
	// this call are archived. Archived events are typically still made
	// available to external event consumers, but will no longer be needed for
	// loading aggregate roots.
	//
	// The events slice may be empty, which allows archiving all existing events
	// without adding any new events.
	WriteEvents(
		ctx context.Context,
		hk, id string,
		end uint64,
		events []*envelopespec.Envelope,
		archive bool,
	) error
}
