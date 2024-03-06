package engineconfig

import (
	"context"
	"fmt"
	"sync"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream"
)

// EventCoordinator is a test implementation for appending and consuming events.
type EventCoordinator struct {
	StreamID *uuidpb.UUID

	mu              sync.RWMutex
	events          []eventstream.Event
	newEventsOffset chan int
}

func NewEventCoordinator(streamID *uuidpb.UUID) *EventCoordinator {
	return &EventCoordinator{
		StreamID:        streamID,
		newEventsOffset: make(chan int, 1),
	}
}

// AppendEvents appends events.
func (c *EventCoordinator) AppendEvents(
	ctx context.Context,
	req eventstream.AppendRequest,
) (eventstream.AppendResponse, error) {
	beginOffset := len(c.events)
	endOffset := beginOffset + 1

	c.mu.Lock()
	for _, env := range req.Events {
		c.events = append(c.events, eventstream.Event{
			StreamID: c.StreamID,
			Offset:   eventstream.Offset(endOffset),
			Envelope: env,
		})
		endOffset++
	}
	c.mu.Unlock()

	if len(req.Events) > 0 {
		c.newEventsOffset <- endOffset
	}

	return eventstream.AppendResponse{
		BeginOffset:            eventstream.Offset(beginOffset),
		EndOffset:              eventstream.Offset(endOffset),
		AppendedByPriorAttempt: false,
	}, nil
}

// SelectEventStream selects an event stream.
func (c *EventCoordinator) SelectEventStream(
	ctx context.Context,
) (streamID *uuidpb.UUID, offset eventstream.Offset, err error) {
	return c.StreamID, eventstream.Offset(c.eventLen()), nil
}

func (c *EventCoordinator) eventLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// Consume consumes from an event stream.
func (c *EventCoordinator) Consume(
	ctx context.Context,
	streamID *uuidpb.UUID,
	offset eventstream.Offset,
	events chan<- eventstream.Event,
) error {
	if c.eventLen() < int(offset) {
		return fmt.Errorf("invalid offset %d", offset)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newOffset := <-c.newEventsOffset:
			c.mu.RLock()
			for _, event := range c.events[(len(c.events)+1)-newOffset:] {
				events <- event
			}
			c.mu.RUnlock()
		}
	}
}
