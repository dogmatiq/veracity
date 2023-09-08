package integration_test

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type eventRecorderStub struct {
	AppendEventsFunc      func(ctx context.Context, req eventstream.AppendRequest) error
	SelectEventStreamFunc func(context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error)
}

func (s *eventRecorderStub) AppendEvents(ctx context.Context, req eventstream.AppendRequest) error {
	if s.AppendEventsFunc != nil {
		return s.AppendEventsFunc(ctx, req)
	}
	return nil
}

func (s *eventRecorderStub) SelectEventStream(ctx context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error) {
	if s.SelectEventStreamFunc != nil {
		return s.SelectEventStreamFunc(ctx)
	}
	return nil, 0, nil
}
