package memory_test

import (
	"context"

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
	})
})
