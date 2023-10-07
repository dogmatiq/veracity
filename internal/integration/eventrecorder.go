package integration

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

// EventRecorder is an interface for appending events to event streams.
type EventRecorder interface {
	AppendEvents(context.Context, eventstream.AppendRequest) (eventstream.AppendResponse, error)
	SelectEventStream(context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error)
}
