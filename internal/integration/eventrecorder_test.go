package integration_test

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type eventRecorderStub struct {
	SelectEventStreamFunc func(context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error)
	AppendEventsFunc      func(ctx context.Context, req eventstream.AppendRequest) (eventstream.AppendResponse, error)
}

func (s *eventRecorderStub) SelectEventStream(ctx context.Context) (streamID *uuidpb.UUID, offset eventstream.Offset, err error) {
	if s.SelectEventStreamFunc != nil {
		return s.SelectEventStreamFunc(ctx)
	}
	return nil, 0, nil
}

func (s *eventRecorderStub) AppendEvents(ctx context.Context, req eventstream.AppendRequest) (eventstream.AppendResponse, error) {
	if s.AppendEventsFunc != nil {
		return s.AppendEventsFunc(ctx, req)
	}
	return eventstream.AppendResponse{}, nil
}
