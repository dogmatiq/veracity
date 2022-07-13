package persistencetest

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/marshalkit/codec"
	"github.com/dogmatiq/marshalkit/codec/json"
	"github.com/dogmatiq/veracity/aggregate"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// AggregateSnapshotContext encapsulates values used during snapshot tests.
type AggregateSnapshotContext struct {
	Reader              aggregate.SnapshotReader
	Writer              aggregate.SnapshotWriter
	ArchiveIsHardDelete bool
	AfterEach           func()
}

// DeclareAggregateSnapshotTests declares a function test-suite for persistence
// of aggregate snapshots.
func DeclareAggregateSnapshotTests(
	new func(marshalkit.ValueMarshaler) AggregateSnapshotContext,
) {
	var (
		tc   AggregateSnapshotContext
		ctx  context.Context
		root *aggregateRoot
	)

	ginkgo.BeforeEach(func() {
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
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		tc = new(m)
	})

	ginkgo.AfterEach(func() {
		if tc.AfterEach != nil {
			tc.AfterEach()
		}
	})

	ginkgo.Describe("func ReadSnapshot()", func() {
		ginkgo.DescribeTable(
			"it populates the root from the latest snapshot",
			func(offset uint64) {
				snapshot := &aggregateRoot{
					Value: fmt.Sprintf("<offset-%d>", offset),
				}

				err := tc.Writer.WriteSnapshot(
					ctx,
					"<handler>",
					"<instance>",
					snapshot,
					offset,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				snapshotOffset, ok, err := tc.Reader.ReadSnapshot(
					ctx,
					"<handler>",
					"<instance>",
					root,
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(snapshotOffset).To(gomega.Equal(offset))
				gomega.Expect(root).To(gomega.Equal(snapshot))
			},
			ginkgo.Entry("at offset 0", uint64(0)),
			ginkgo.Entry("at offset 1", uint64(1)),
			ginkgo.Entry("at offset 2", uint64(2)),
		)

		ginkgo.It("stores separate snapshots for each combination of handler key and instance ID", func() {
			type snapshotKey struct {
				HandlerKey string
				InstanceID string
			}

			instances := []snapshotKey{
				{"<handler-1>", "<instance-1>"},
				{"<handler-1>", "<instance-2>"},
				{"<handler-2>", "<instance-1>"},
				{"<handler-2>", "<instance-2>"},
			}

			for i, inst := range instances {
				err := tc.Writer.WriteSnapshot(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					&aggregateRoot{
						Value: inst.HandlerKey + inst.InstanceID,
					},
					uint64(i),
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			for i, inst := range instances {
				snapshotOffset, ok, err := tc.Reader.ReadSnapshot(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					root,
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(snapshotOffset).To(gomega.BeNumerically("==", i))
				gomega.Expect(root.Value).To(gomega.Equal(inst.HandlerKey + inst.InstanceID))
			}
		})

		ginkgo.It("does not modify the root when there are no snapshots", func() {
			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeFalse())
			gomega.Expect(root.Value).To(gomega.BeEmpty())
		})

		ginkgo.It("does not modify the root when the latest snapshot is older than minOffset", func() {
			snapshot := &aggregateRoot{}

			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				10,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				11,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeFalse())
			gomega.Expect(root.Value).To(gomega.BeEmpty())
		})

		ginkgo.It("populates the root when the latest snapshot is taken at exactly minOffset", func() {
			snapshot := &aggregateRoot{
				Value: "<snapshot>",
			}
			expectedSnapshotOffset := uint64(10)

			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				expectedSnapshotOffset,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			snapshotOffset, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				expectedSnapshotOffset,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(snapshotOffset).To(gomega.Equal(expectedSnapshotOffset))
			gomega.Expect(root).To(gomega.Equal(snapshot))
		})

		ginkgo.It("does not modify the root when there are no compatible snapshots", func() {
			snapshot := &incompatibleAggregateRoot{}

			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeFalse())
			gomega.Expect(root.Value).To(gomega.BeEmpty())
		})

		ginkgo.It("does not retain a reference to the original snapshot value", func() {
			snapshot := &aggregateRoot{
				Value: "<original>",
			}

			ginkgo.By("persisting a snapshot")
			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				snapshot,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("modifying the original snapshot value after it was persisted")
			snapshot.Value = "<changed>"

			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(root.Value).To(gomega.Equal("<original>"))
		})

		ginkgo.It("does not retain a reference to the loaded snapshot value", func() {
			originalSnapshot := &aggregateRoot{
				Value: "<original>",
			}

			ginkgo.By("persisting a snapshot")
			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				originalSnapshot,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("reading the snapshot")
			loadedSnapshot := &aggregateRoot{}

			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				loadedSnapshot,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeTrue())

			ginkgo.By("modifying the loaded snapshot")
			loadedSnapshot.Value = "<changed>"

			ginkgo.By("reading the snapshot again")
			_, ok, err = tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(root.Value).To(gomega.Equal("<original>"))
		})

		ginkgo.It("does not find archived snapshots", func() {
			err := tc.Writer.WriteSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				&aggregateRoot{
					Value: "<snapshot>",
				},
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.ArchiveSnapshots(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			if !tc.ArchiveIsHardDelete {
				ginkgo.Skip("archived snapshots are not hard deleted")
			}

			_, ok, err := tc.Reader.ReadSnapshot(
				ctx,
				"<handler>",
				"<instance>",
				root,
				0,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(ok).To(gomega.BeFalse())
			gomega.Expect(root.Value).To(gomega.BeEmpty())
		})
	})

	ginkgo.Describe("func ArchiveSnapshots()", func() {
		ginkgo.It("does not archive snapshots of other instances", func() {
			type snapshotKey struct {
				HandlerKey string
				InstanceID string
			}

			instances := []snapshotKey{
				{"<handler-1>", "<instance-1>"},
				{"<handler-1>", "<instance-2>"},
				{"<handler-2>", "<instance-1>"},
				{"<handler-2>", "<instance-2>"},
			}
			archiveKey := instances[0]

			for i, inst := range instances {
				err := tc.Writer.WriteSnapshot(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					&aggregateRoot{
						Value: inst.HandlerKey + inst.InstanceID,
					},
					uint64(i),
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			err := tc.Writer.ArchiveSnapshots(
				ctx,
				archiveKey.HandlerKey,
				archiveKey.InstanceID,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			for i, inst := range instances {
				if inst == archiveKey {
					continue
				}

				snapshotOffset, ok, err := tc.Reader.ReadSnapshot(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					root,
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(snapshotOffset).To(gomega.BeNumerically("==", i))
				gomega.Expect(root.Value).To(gomega.Equal(inst.HandlerKey + inst.InstanceID))
			}
		})

		ginkgo.It("does not return an error if the snapshot does not exist", func() {
			err := tc.Writer.ArchiveSnapshots(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})
}

type aggregateRoot struct {
	dogma.AggregateRoot

	Value string
}

type incompatibleAggregateRoot struct {
	dogma.AggregateRoot
}
