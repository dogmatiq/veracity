package memory

import (
	"context"
	"sync"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// AggregateEventStore stores event messages produced by aggregates
type AggregateEventStore struct {
	m      sync.RWMutex
	events map[instanceKey]instanceEvents
}

type instanceEvents struct {
	FirstOffset uint64
	NextOffset  uint64
}

// ReadBounds returns the offsets that are the bounds of the relevant historical
// events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// firstOffset is the offset of the first event recorded by the instance that is
// considered relevant to its current state. It begins as zero, but is advanced
// to the offset after the last archived event when events are archived.
//
// nextOffset is the offset that will be occupied by the next event to be
// recorded by this instance.
func (s *AggregateEventStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (firstOffset, nextOffset uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	e := s.events[k]

	return e.FirstOffset, e.NextOffset, nil
}

// ReadEvents loads some historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// If more is true there are more events to be loaded, and ReadEvents() should
// be called again with firstOffset incremented by len(events).
//
// If more is false there are no subsequent historical events to be loaded.
//
// The maximum number of events returned by each call is implementation defined.
//
// It may return an error if firstOffset refers to an archived event.
func (s *AggregateEventStore) ReadEvents(
	ctx context.Context,
	hk, id string,
	firstOffset uint64,
) (events []*envelopespec.Envelope, more bool, _ error) {
	return nil, false, nil
}

// WriteEvents writes events that were recorded by an aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// nextOffset must be the offset immediately after the offset of the last event
// written; otherwise, no events are recorded and an error is returned.
//
// If archive is true, all prior events and the events being written by this
// call are archived. Archived events are typically still made available to
// external event consumers, but will no longer be needed for loading aggregate
// roots.
func (s *AggregateEventStore) WriteEvents(
	ctx context.Context,
	hk, id string,
	nextOffset uint64,
	events []*envelopespec.Envelope,
	archive bool,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	e := s.events[k]

	e.NextOffset += uint64(len(events))

	if archive {
		e.FirstOffset = e.NextOffset
	}

	if s.events == nil {
		s.events = map[instanceKey]instanceEvents{}
	}

	s.events[k] = e

	return nil
}
