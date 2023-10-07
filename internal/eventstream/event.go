package eventstream

import (
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// Offset is the offset of an event within a stream.
type Offset uint64

// Event is an event message that has been recorded on a stream.
type Event struct {
	StreamID *uuidpb.UUID
	Offset   Offset
	Envelope *envelopepb.Envelope
}
