package integration_test

import "github.com/dogmatiq/enginekit/protobuf/envelopepb"

type eventRecorderStub struct {
	RecordEventsFunc func(events []*envelopepb.Envelope) error
}

func (s *eventRecorderStub) RecordEvents(events []*envelopepb.Envelope) error {
	if s.RecordEventsFunc != nil {
		return s.RecordEventsFunc(events)
	}
	return nil
}
