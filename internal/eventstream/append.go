package eventstream

import (
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// AppendRequest is a request to append events to an event stream.
type AppendRequest struct {
	// StreamID is the ID of the event stream to which the events are appended.
	StreamID *uuidpb.UUID

	// Events is the set of events to append to the stream. It must not be
	// empty.
	Events []*envelopepb.Envelope

	// LowestPossibleOffset is the lowest offset within the stream at which
	// these events may have already been appended.
	LowestPossibleOffset Offset
}

// AppendResponse is the successful result of an [AppendRequest].
type AppendResponse struct {
	// [BeginOffset, EndOffset) is the half-open range describing the offsets of the
	// appended events within the stream. That is, BeginOffset is the offset of the
	// first event in the [AppendRequest], and EndOffset is the offset after the last
	// event in the [AppendRequest].
	BeginOffset, EndOffset Offset

	// AppendedByPriorAttempt is true if the events were appended by a prior
	// [AppendRequest] and hence deduplicated.
	AppendedByPriorAttempt bool
}
