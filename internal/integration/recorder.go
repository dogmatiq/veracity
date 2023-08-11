package integration

import (
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type EventRecorder interface {
	AppendEvents(req eventstream.AppendRequest) error
}
