package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

type EventConsumer struct {
	// TODO: add an ability to read historical events from supervisor journals
	// along with the ability to current events from Events channel.
	Events chan Event // eventLen func
}

func (c *EventConsumer) Consume(
	ctx context.Context,
	streamID *uuidpb.UUID,
	offset Offset,
	events chan<- Event,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-c.Events:
			events <- e
		}
	}
}
