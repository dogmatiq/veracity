package integration

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type EventRecorder interface {
	AppendEvents(context.Context, eventstream.AppendRequest) error
	SelectEventStream(context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error)
}
