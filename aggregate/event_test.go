package aggregate_test

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/gomega"
)

// eventReaderStub is a test implementation of the EventReader interface.
type eventReaderStub struct {
	EventReader

	ReadBoundsFunc func(
		ctx context.Context,
		hk, id string,
	) (begin, end uint64, _ error)

	ReadEventsFunc func(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (events []*envelopespec.Envelope, end uint64, _ error)
}

func (s *eventReaderStub) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, end uint64, _ error) {
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
	begin uint64,
) (events []*envelopespec.Envelope, end uint64, _ error) {
	if s.ReadEventsFunc != nil {
		return s.ReadEventsFunc(ctx, hk, id, begin)
	}

	if s.EventReader != nil {
		return s.EventReader.ReadEvents(ctx, hk, id, begin)
	}

	return nil, begin, nil
}

// eventWriterStub is a test implementation of the EventWriter interface.
type eventWriterStub struct {
	EventWriter

	WriteEventsFunc func(
		ctx context.Context,
		hk, id string,
		end uint64,
		events []*envelopespec.Envelope,
		archive bool,
	) error
}

func (s *eventWriterStub) WriteEvents(
	ctx context.Context,
	hk, id string,
	end uint64,
	events []*envelopespec.Envelope,
	archive bool,
) error {
	if s.WriteEventsFunc != nil {
		return s.WriteEventsFunc(ctx, hk, id, end, events, archive)
	}

	if s.EventWriter != nil {
		return s.EventWriter.WriteEvents(ctx, hk, id, end, events, archive)
	}

	return nil
}

// expectEvents reads all events from store starting and the given beginning
// revision and asserts that they are equal to expectedEvents.
func expectEvents(
	reader EventReader,
	hk, id string,
	begin uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, end, err := reader.ReadEvents(
			context.Background(),
			hk,
			id,
			begin,
		)
		Expect(err).ShouldNot(HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if begin >= end {
			break
		}

		begin = end
	}

	if len(producedEvents) == 0 && len(expectedEvents) == 0 {
		return
	}

	Expect(producedEvents).To(EqualX(expectedEvents))
}
