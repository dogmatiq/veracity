package integration_test

import "github.com/dogmatiq/enginekit/protobuf/envelopepb"

type eventRecorderStub struct {
	RecordEventsFunc func(events []*envelopepb.Envelope)
}

func (s *eventRecorderStub) RecordEvents(events []*envelopepb.Envelope) {
	if s.RecordEventsFunc != nil {
		s.RecordEventsFunc(events)
	}
}
