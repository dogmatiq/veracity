package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

type EventConsumer struct {
	// eventLen func
}

func (c *EventConsumer) Consume(
	ctx context.Context,
	streamID *uuidpb.UUID,
	offset Offset,
	events chan<- Event,
) error {
	// if c.eventLen() < int(offset) {
	// 	return fmt.Errorf("invalid offset %d", offset)
	// }

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return ctx.Err()
	// 	case newOffset := <-c.newEventsOffset:
	// 		c.mu.RLock()
	// 		for _, event := range c.events[(len(c.events)+1)-newOffset:] {
	// 			events <- event
	// 		}
	// 		c.mu.RUnlock()
	// 	}
	// }
	return nil
}
