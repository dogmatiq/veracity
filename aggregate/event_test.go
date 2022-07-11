package aggregate_test

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/aggregate"
)

// eventReaderStub is a test implementation of the EventReader interface.
type eventReaderStub struct {
	EventReader

	ReadBoundsFunc func(
		ctx context.Context,
		hk, id string,
	) (firstOffset, nextOffset uint64, _ error)

	ReadEventsFunc func(
		ctx context.Context,
		hk, id string,
		firstOffset uint64,
	) (events []*envelopespec.Envelope, more bool, _ error)
}

func (s *eventReaderStub) ReadBounds(
	ctx context.Context,
	hk, id string,
) (firstOffset, nextOffset uint64, _ error) {
	if s.ReadBoundsFunc != nil {
		return s.ReadBoundsFunc(ctx, hk, id)
	}

	if s.EventReader != nil {
		return s.EventReader.ReadBounds(ctx, hk, id)
	}

	return 0, 0, nil
}

func (s *eventReaderStub) ReadEvents(
	ctx context.Context,
	hk, id string,
	firstOffset uint64,
) (events []*envelopespec.Envelope, more bool, _ error) {
	if s.ReadEventsFunc != nil {
		return s.ReadEventsFunc(ctx, hk, id, firstOffset)
	}

	if s.EventReader != nil {
		return s.EventReader.ReadEvents(ctx, hk, id, firstOffset)
	}

	return nil, false, nil
}

// eventWriterStub is a test implementation of the EventWriter interface.
type eventWriterStub struct {
	EventWriter

	WriteEventsFunc func(
		ctx context.Context,
		hk, id string,
		firstOffset, nextOffset uint64,
		events []*envelopespec.Envelope,
		archive bool,
	) error
}

func (s *eventWriterStub) WriteEvents(
	ctx context.Context,
	hk, id string,
	firstOffset, nextOffset uint64,
	events []*envelopespec.Envelope,
	archive bool,
) error {
	if s.WriteEventsFunc != nil {
		return s.WriteEventsFunc(ctx, hk, id, firstOffset, nextOffset, events, archive)
	}

	if s.EventWriter != nil {
		return s.EventWriter.WriteEvents(ctx, hk, id, firstOffset, nextOffset, events, archive)
	}

	return nil
}
