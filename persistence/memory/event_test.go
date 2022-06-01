package memory_test

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type AggregateEventStore", func() {
	var store *AggregateEventStore

	BeforeEach(func() {
		store = &AggregateEventStore{}
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
				[]*envelopespec.Envelope{
					{},
					{},
					{},
				},
				false,
			)

			Expect(err).ShouldNot(HaveOccurred())

			_, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(nextOffset).To(BeNumerically("==", 3))
		})

		It("returns the next unused offset after several writes", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				[]*envelopespec.Envelope{
					{},
					{},
					{},
				},
				false,
			)

			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				[]*envelopespec.Envelope{
					{},
					{},
				},
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
				[]*envelopespec.Envelope{
					{},
					{},
					{},
				},
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
				[]*envelopespec.Envelope{
					{},
					{},
					{},
				},
				true,
			)

			Expect(err).ShouldNot(HaveOccurred())

			firstOffset, nextOffset, err := store.ReadBounds(
				context.Background(),
				"<handler>",
				"<instance>",
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(firstOffset).To(BeNumerically("==", 3))
			Expect(nextOffset).To(BeNumerically("==", 3))
		})

		It("returns a firstOffset equal to the offset of the first unarchived event", func() {
			err := store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				[]*envelopespec.Envelope{
					{},
					{},
					{},
				},
				true,
			)

			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteEvents(
				context.Background(),
				"<handler>",
				"<instance>",
				0,
				[]*envelopespec.Envelope{
					{},
					{},
				},
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
	})
})
