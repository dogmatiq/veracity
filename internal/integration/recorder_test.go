package integration_test

import (
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type eventRecorderStub struct {
	AppendEventsFunc func(req eventstream.AppendRequest) error
}

func (s *eventRecorderStub) AppendEvents(req eventstream.AppendRequest) error {
	if s.AppendEventsFunc != nil {
		return s.AppendEventsFunc(req)
	}
	return nil
}
