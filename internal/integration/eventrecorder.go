package integration

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

// EventRecorder is an interface for appending events produced by a
// [dogma.IntegrationMessageHandler] to event streams.
type EventRecorder interface {
	// SelectEventStream returns the ID and next offset of the event stream to
	// which new events should be appended.
	SelectEventStream(context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error)

	// AppendEvents appends events to a stream.
	AppendEvents(context.Context, eventstream.AppendRequest) (eventstream.AppendResponse, error)
}
