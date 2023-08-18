package integration

import (
	"context"

	"github.com/dogmatiq/veracity/internal/eventstream"
)

type EventRecorder interface {
	AppendEvents(context.Context, eventstream.AppendRequest) error
}
