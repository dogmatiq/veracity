package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregatejournaled"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/gomega"
)

type eventStreamStub struct {
	EventStream

	WriteFunc         func(ctx context.Context, ev parcel.Parcel) error
	WriteAtOffsetFunc func(ctx context.Context, offset uint64, ev parcel.Parcel) error
}

func (s *eventStreamStub) Read(
	ctx context.Context,
	offset uint64,
) ([]parcel.Parcel, error) {
	return s.EventStream.Read(ctx, offset)
}

func (s *eventStreamStub) Write(
	ctx context.Context,
	ev parcel.Parcel,
) error {
	if s.WriteFunc != nil {
		return s.WriteFunc(ctx, ev)
	}

	return s.EventStream.Write(ctx, ev)
}

func (s *eventStreamStub) WriteAtOffset(
	ctx context.Context,
	offset uint64,
	ev parcel.Parcel,
) error {
	if s.WriteAtOffsetFunc != nil {
		return s.WriteAtOffsetFunc(ctx, offset, ev)
	}

	return s.EventStream.WriteAtOffset(ctx, offset, ev)
}

// expectEvents reads all events from an EventStore starting at the given offset
// and asserts that they are equal to the expected events.
func expectEvents(
	ctx context.Context,
	s EventStream,
	offset uint64,
	expected ...parcel.Parcel,
) {
	actual := readAllEvents(ctx, s, offset)

	if len(actual) == 0 && len(expected) == 0 {
		return
	}

	ExpectWithOffset(1, actual).To(EqualX(expected))
}

// readAllEvents reads all events from an EventStore starting at the given
// offset.
func readAllEvents(
	ctx context.Context,
	s EventStream,
	offset uint64,
) []parcel.Parcel {
	var result []parcel.Parcel

	for {
		events, err := s.Read(ctx, offset)
		ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

		if len(events) == 0 {
			break
		}

		result = append(result, events...)
		offset += uint64(len(events))
	}

	return result
}
