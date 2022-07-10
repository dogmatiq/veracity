package memory_test

import (
	"context"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	_ aggregate.EventReader = (*AggregateEventStore)(nil)
	_ aggregate.EventWriter = (*AggregateEventStore)(nil)
)

var _ = Describe("type AggregateEventStore", func() {
	var store *AggregateEventStore
	var allEvents []*envelopespec.Envelope

	BeforeEach(func() {
		store = &AggregateEventStore{}
		allEvents = []*envelopespec.Envelope{
			NewEnvelope("<event-0>", MessageA1),
			NewEnvelope("<event-1>", MessageB1),
			NewEnvelope("<event-2>", MessageC1),
			NewEnvelope("<event-4>", MessageD1),
			NewEnvelope("<event-5>", MessageE1),
		}
	})

	Describe("func ReadBounds()", func() {
		It("returns zero offsets when there are no historical events", func() {
			firstOffset, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 0))
			Expect(nextOffset).To(BeNumerically("==", 0))
		})

		It("returns the next unused offset", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nextOffset).To(BeNumerically("==", 5))
		})

		It("returns the next unused offset after several writes", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				3,
				allEvents[3:],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nextOffset).To(BeNumerically("==", 5))
		})

		It("returns a zero firstOffset when there are historical events", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			firstOffset, _, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 0))
		})

		It("returns a firstOffset equal to nextOffset when all events are archived", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				3,
				allEvents[3:],
				true, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			firstOffset, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 5))
			Expect(nextOffset).To(BeNumerically("==", 5))
		})

		It("returns a firstOffset equal to the offset of the first unarchived event", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				true, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				3,
				3,
				allEvents[3:],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			firstOffset, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 3))
			Expect(nextOffset).To(BeNumerically("==", 5))
		})
	})

	Describe("func ReadEvents()", func() {
		It("produces the events in the order they were written", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				3,
				allEvents[3:],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())
			expectEvents(
				store,
				"<handler>",
				"<instance>",
				0,
				allEvents,
			)
		})

		It("produces no events when there are no historical events", func() {
			events, more, err := store.ReadEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(events).To(BeEmpty())
			Expect(more).To(BeFalse())
		})

		It("produces no events when the offset is larger than the offset of the most recent event", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			events, more, err := store.ReadEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				5,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(events).To(BeEmpty())
			Expect(more).To(BeFalse())
		})

		It("returns an error when the offset refers to an archived event", func() {
			// Note this is specific to the memory implementation

			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				true, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, _, err = store.ReadEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
			)
			Expect(err).To(MatchError("event at offset 0 is archived"))
		})

		It("does not return an error when the offset refers to an event after the last archived event", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				true, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				3,
				3,
				allEvents[3:],
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())
			expectEvents(
				store,
				"<handler>",
				"<instance>",
				3,
				allEvents[3:],
			)
		})

		It("returns an error if the offset is greater than the next offset", func() {
			_, _, err := store.ReadEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				2,
			)
			Expect(err).To(MatchError("event at offset 2 does not exist yet"))
		})
	})

	Describe("func WriteEvents()", func() {
		It("returns an error if firstOffset is not the actual first offset", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				3, // incorrect firstOffset
				0,
				allEvents,
				false, // archive
			)
			Expect(err).To(MatchError("optimistic concurrency conflict, 3 is not the first offset"))
		})

		It("returns an error if nextOffset is not the actual next offset", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				3, // incorrect nextOffset
				allEvents,
				false, // archive
			)
			Expect(err).To(MatchError("optimistic concurrency conflict, 3 is not the next offset"))
		})

		It("allows archiving without adding more events", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				5,
				nil, // don't add more events
				true,
			)
			Expect(err).ShouldNot(HaveOccurred())

			firstOffset, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 5))
			Expect(nextOffset).To(BeNumerically("==", 5))
		})

		It("stores separate bounds for each combination of handler key and instance ID", func() {
			type instanceKey struct {
				HandlerKey string
				InstanceID string
			}

			instances := []instanceKey{
				{"<handler-1>", "<instance-1>"},
				{"<handler-1>", "<instance-2>"},
				{"<handler-2>", "<instance-1>"},
				{"<handler-2>", "<instance-2>"},
			}

			for i, inst := range instances {
				events := allEvents[:i+1]

				err := store.WriteEvents(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
					0,
					0,
					events,
					true, // archive
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = store.WriteEvents(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
					uint64(len(events)),
					uint64(len(events)),
					events,
					false, // archive
				)
				Expect(err).ShouldNot(HaveOccurred())
			}

			for i, inst := range instances {
				firstOffset, nextOffset, err := store.ReadBounds(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
				)

				expectedFirstOffset := i + 1
				expectedNextOffset := expectedFirstOffset + i + 1

				Expect(err).ShouldNot(HaveOccurred())
				Expect(firstOffset).To(BeNumerically("==", expectedFirstOffset))
				Expect(nextOffset).To(BeNumerically("==", expectedNextOffset))
			}
		})

		It("stores separate events for each combination of handler key and instance ID", func() {
			type instanceKey struct {
				HandlerKey string
				InstanceID string
			}

			instances := []instanceKey{
				{"<handler-1>", "<instance-1>"},
				{"<handler-1>", "<instance-2>"},
				{"<handler-2>", "<instance-1>"},
				{"<handler-2>", "<instance-2>"},
			}

			for i, inst := range instances {
				events := allEvents[:i+1]

				err := store.WriteEvents(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
					0,
					0,
					events,
					true, // archive
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = store.WriteEvents(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
					uint64(len(events)),
					uint64(len(events)),
					events,
					false, // archive
				)
				Expect(err).ShouldNot(HaveOccurred())
			}

			for i, inst := range instances {
				expectEvents(
					store,
					inst.HandlerKey,
					inst.InstanceID,
					uint64(i+1),
					allEvents[:i+1],
				)
			}
		})
	})
})

// expectEvents reads all events from store starting from offset and asserts
// that they are equal to expectedEvents.
func expectEvents(
	store *AggregateEventStore,
	hk, id string,
	offset uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, more, err := store.ReadEvents(
			context.Background(),
			hk,
			id,
			offset,
		)
		Expect(err).ShouldNot(HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if !more {
			break
		}

		offset += uint64(len(events))
	}

	Expect(producedEvents).To(EqualX(expectedEvents))
}
