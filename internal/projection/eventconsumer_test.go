package projection_test

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type eventConsumerStub struct {
	ConsumeFunc func(ctx context.Context, streamID *uuidpb.UUID, offset eventstream.Offset, events chan<- eventstream.Event) error
}

func (s *eventConsumerStub) Consume(ctx context.Context, streamID *uuidpb.UUID, offset eventstream.Offset, events chan<- eventstream.Event) error {
	if s.ConsumeFunc != nil {
		return s.ConsumeFunc(ctx, streamID, offset, events)
	}
	return nil
}
