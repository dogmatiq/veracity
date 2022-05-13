package aggregate

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// EventReader is an interface for reading information about events that were
// recorded as a result of sending command messages to aggregates.
type EventReader interface {
	// ReadBounds returns the offset of the first unarchived event, and the next
	// "unused" offset for a specific specific aggregate instance.
	ReadBounds(
		ctx context.Context,
		handlerKey string,
		instanceID string,
	) (firstOffset, nextOffset uint64, _ error)

	// ReadEvents loads historical events for a specific aggregate instance.
	//
	// The number of loaded events is implementation defined. All events have
	// been loaded when the returned slice is empty.
	//
	// It returns an error if firstOffset refers to an archived event.
	ReadEvents(
		ctx context.Context,
		handlerKey string,
		instanceID string,
		firstOffset uint64,
	) ([]*envelopespec.Envelope, error)
}

// EventWriter is an interface for writing events produced by an aggregate to
// persistent storage.
type EventWriter interface {
	// WriteEvents write events to an aggregate instances event stream.
	//
	// nextOffset must be the offset immediately after the offset of the last
	// event written; otherwise, an error is returned.
	//
	// If archive is true, all historical events and the events being written by
	// this call are archived. Archived events are still available to external
	// consumers, but are no longer loaded when reproducing aggregate root
	// state.
	WriteEvents(
		ctx context.Context,
		handlerKey, instanceID string,
		nextOffset uint64,
		events []*envelopespec.Envelope,
		archive bool,
	) error
}
