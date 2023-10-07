package eventstream

import (
	"reflect"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// StreamAvailable is an engine event that indicates that an event stream has
// been discovered and is available for use.
type StreamAvailable struct {
	StreamID   *uuidpb.UUID
	EventTypes []reflect.Type
}

// StreamUnavailable is an engine event that indicates that an event stream is
// no longer available for use.
type StreamUnavailable struct {
	StreamID *uuidpb.UUID
}
