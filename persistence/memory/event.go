package memory

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// AggregateEventStore stores event messages produced by aggregates
type AggregateEventStore struct {
}

// ReadBounds returns "bounds" of the events for a specific aggregate
// instance.
//
// hk is the identity key of the aggregate message handler. id is the
// aggregate instance ID.
//
// firstOffset is the offset of the first event recorded by the instance
// that is considered relevant to its current state. It begins as zero, but
// is advanced to the offset after the last archived event when events are
// archived.
//
// nextOffset is the offset that will be occupied by the next event to be
// recorded by this instance.
func (s *AggregateEventStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (firstOffset, nextOffset uint64, _ error) {
	return 0, 0, nil
}

// ReadEvents loads some historical events for a specific aggregate
// instance.
//
// hk is the identity key of the aggregate message handler. id is the
// aggregate instance ID.
//
// If more is true there are more events to be loaded, and ReadEvents()
// should be called again with firstOffset incremented by len(events).
//
// If more is false there are no subsequent historical events to be loaded.
//
// The maximum number of events returned by each call is implementation
// defined.
//
// It may return an error if firstOffset refers to an archived event.
func (s *AggregateEventStore) ReadEvents(
	ctx context.Context,
	hk, id string,
	firstOffset uint64,
) (events []*envelopespec.Envelope, more bool, _ error) {
	panic("not implemented")
}
