package memory_test

import (
	"context"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("type AggregateSnapshotStore", func() {

	var (
		ctx   context.Context
		root  *AggregateRoot
		store *AggregateSnapshotStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		root = &AggregateRoot{}
		store = &AggregateSnapshotStore{}
	})

	Describe("func ReadSnapshot()", func() {
		DescribeTable(
			"it populates the root from the latest snapshot",
			func(events []dogma.Message) {
				snapshot := &AggregateRoot{}

				for _, ev := range events {
					snapshot.ApplyEvent(ev)
				}

				expectedSnapshotOffset := uint64(len(events) - 1)
				err := store.WriteSnapshot(
					ctx,
					"<handler>",
					"<instance>",
					snapshot,
					expectedSnapshotOffset,
				)
				Expect(err).ShouldNot(HaveOccurred())

				snapshotOffset, ok, err := store.ReadSnapshot(
					ctx,
					"<handler>",
					"<instance>",
					root,
					0,
				)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(snapshotOffset).To(Equal(expectedSnapshotOffset))
				Expect(root).To(Equal(snapshot))
			},
			Entry("at offset 0", []dogma.Message{MessageE1}),
			Entry("at offset 1", []dogma.Message{MessageE1, MessageE2}),
			Entry("at offset 2", []dogma.Message{MessageE1, MessageE2, MessageE3}),
		)

		It("does not modify the root when there are no snapshots", func() {
			_, ok, err := store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(root.AppliedEvents).To(BeEmpty())
		})
	})
})
