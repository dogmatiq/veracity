package integration_test

import (
	"context"

	"github.com/dogmatiq/veracity/internal/eventstream"
)

type eventRecorderStub struct {
	AppendEventsFunc func(ctx context.Context, req eventstream.AppendRequest) error
}

func (s *eventRecorderStub) AppendEvents(ctx context.Context, req eventstream.AppendRequest) error {
	if s.AppendEventsFunc != nil {
		return s.AppendEventsFunc(ctx, req)
	}
	return nil
}
