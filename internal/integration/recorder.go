package integration

import "github.com/dogmatiq/enginekit/protobuf/envelopepb"

type EventRecorder interface {
	RecordEvents(events []*envelopepb.Envelope) error
}
