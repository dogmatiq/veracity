package projection

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

type EventConsumer interface {
	Consume(
		ctx context.Context,
		streamID *uuidpb.UUID,
		offset eventstream.Offset,
		events chan<- eventstream.Event,
	) error
}
