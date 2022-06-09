package memory_test

import (
	"context"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
				allEvents,
				false,
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
				allEvents[:3],
				false,
			)

			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[3:],
				false,
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
				allEvents,
				false,
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
				allEvents[:3],
				false,
			)

			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[3:],
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

		It("returns a firstOffset equal to the offset of the first unarchived event", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[:3],
				true,
			)

			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[3:],
				false,
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
					events,
					true,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = store.WriteEvents(
					context.Background(),
					inst.HandlerKey,
					inst.InstanceID,
					0,
					events,
					false,
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
	})

	Describe("func ReadEvents()", func() {
		It("produces the events in the order they were written", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[:3],
				false,
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				allEvents[3:],
				false,
			)
			Expect(err).ShouldNot(HaveOccurred())

			var producedEvents []*envelopespec.Envelope
			for {
				events, more, err := store.ReadEvents(
					context.Background(),
					"<handler>",
					"<instance>",
					0,
				)

				Expect(err).ShouldNot(HaveOccurred())

				producedEvents = append(producedEvents, events...)

				if !more {
					break
				}
			}

			Expect(producedEvents).To(Equal(allEvents))
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
				allEvents,
				false,
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
				allEvents,
				true,
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

		XIt("does not return an error when the offset refers to an event after the last archived event", func() {
		})
	})
})
