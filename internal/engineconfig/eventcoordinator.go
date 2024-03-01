package engineconfig

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

// EventCoordinator is a test implementation for appending and consuming events.
type EventCoordinator struct{}

// AppendEvents appends events.
func (c *EventCoordinator) AppendEvents(
	ctx context.Context,
	req eventstream.AppendRequest,
) (eventstream.AppendResponse, error) {
}

// SelectEventStream selects an event stream.
func (c *EventCoordinator) SelectEventStream(
	ctx context.Context,
) (streamID *uuidpb.UUID, offset eventstream.Offset, err error) {
	return uuidpb.Generate(), 0, nil
}

// Consume consumes from an event stream.
func (c *EventCoordinator) Consume(
	ctx context.Context,
	streamID *uuidpb.UUID,
	offset eventstream.Offset,
	events chan<- eventstream.Event,
) error {
	panic("not implemented")
}
