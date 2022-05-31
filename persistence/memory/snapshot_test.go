package memory_test

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit/codec"
	"github.com/dogmatiq/marshalkit/codec/json"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type aggregateRoot struct {
	dogma.AggregateRoot

	Value string
}

type incompatibleAggregateRoot struct {
	dogma.AggregateRoot
}

var _ = Describe("type AggregateSnapshotStore", func() {
	var (
		ctx   context.Context
		root  *aggregateRoot
		store *AggregateSnapshotStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		root = &aggregateRoot{}

		m, err := codec.NewMarshaler(
			[]reflect.Type{
				reflect.TypeOf(&aggregateRoot{}),
				reflect.TypeOf(&incompatibleAggregateRoot{}),
			},
			[]codec.Codec{
				&json.Codec{},
			},
		)
		Expect(err).ShouldNot(HaveOccurred())

		store = &AggregateSnapshotStore{
			Marshaler: m,
		}
	})

	Describe("func ReadSnapshot()", func() {
		DescribeTable(
			"it populates the root from the latest snapshot",
			func(offset uint64) {
				snapshot := &aggregateRoot{
					Value: fmt.Sprintf("<offset-%d>", offset),
				}

				err := store.WriteSnapshot(
					ctx,
					"<handler>",
					"<instance>",
					snapshot,
					offset,
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
				Expect(snapshotOffset).To(Equal(offset))
				Expect(root).To(Equal(snapshot))
			},
			Entry("at offset 0", uint64(0)),
			Entry("at offset 1", uint64(1)),
			Entry("at offset 2", uint64(2)),
		)

		It("stores separate snapshots for each handler", func() {
			err := store.WriteSnapshot(
				ctx,
				"<handler-1>",
				"<instance>",
				&aggregateRoot{
					Value: "<handler-1-snapshot>",
				},
				10,
			)
			Expect(err).ShouldNot(HaveOccurred())

			err = store.WriteSnapshot(
				ctx,
				"<handler-2>",
				"<instance>",
				&aggregateRoot{
					Value: "<handler-2-snapshot>",
				},
				20,
			)
			Expect(err).ShouldNot(HaveOccurred())

			snapshotOffset, ok, err := store.ReadSnapshot(
				ctx,
				"<handler-1>",
				"<instance>",
				root,
				0,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(snapshotOffset).To(BeNumerically("==", 10))
			Expect(root.Value).To(Equal("<handler-1-snapshot>"))

			snapshotOffset, ok, err = store.ReadSnapshot(
				ctx,
				"<handler-2>",
				"<instance>",
				root,
				0,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(snapshotOffset).To(BeNumerically("==", 20))
			Expect(root.Value).To(Equal("<handler-2-snapshot>"))
		})

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
			Expect(root.Value).To(BeEmpty())
		})

		It("does not modify the root when the latest snapshot is older than minOffset", func() {
			snapshot := &aggregateRoot{}

			err := store.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				10,
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, ok, err := store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				11,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(root.Value).To(BeEmpty())
		})

		It("populates the root when the latest snapshot is taken at exactly minOffset", func() {
			snapshot := &aggregateRoot{
				Value: "<snapshot>",
			}
			expectedSnapshotOffset := uint64(10)

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
				expectedSnapshotOffset,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(snapshotOffset).To(Equal(expectedSnapshotOffset))
			Expect(root).To(Equal(snapshot))
		})

		It("does not modify the root when there are no compatible snapshots", func() {
			snapshot := &incompatibleAggregateRoot{}

			err := store.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, ok, err := store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(root.Value).To(BeEmpty())
		})

		It("does not retain a reference to the original snapshot value", func() {
			snapshot := &aggregateRoot{
				Value: "<original>",
			}

			By("persisting a snapshot")
			err := store.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())

			By("modifying the original snapshot value after it was persisted")
			snapshot.Value = "<changed>"

			_, ok, err := store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(root.Value).To(Equal("<original>"))
		})

		It("does not retain a reference to the loaded snapshot value", func() {
			originalSnapshot := &aggregateRoot{
				Value: "<original>",
			}

			By("persisting a snapshot")
			err := store.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				originalSnapshot,
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())

			By("reading the snapshot")
			loadedSnapshot := &aggregateRoot{}

			_, ok, err := store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				loadedSnapshot,
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			By("modifying the loaded snapshot")
			loadedSnapshot.Value = "<changed>"

			By("reading the snapshot again")
			_, ok, err = store.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(root.Value).To(Equal("<original>"))
		})
	})
})
