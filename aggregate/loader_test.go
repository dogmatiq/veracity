package aggregate_test

import (
	"context"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Loader", func() {
	var (
		handlerID configkit.Identity
		events    *memory.AggregateEventStore
		snapshots *memory.AggregateSnapshotStore
		root      *AggregateRoot
		loader    *Loader
	)

	BeforeEach(func() {
		handlerID = configkit.MustNewIdentity("<handler-name>", "<handler>")

		events = &memory.AggregateEventStore{}

		snapshots = &memory.AggregateSnapshotStore{
			Marshaler: Marshaler,
		}

		loader = &Loader{
			EventReader:    events,
			SnapshotWriter: snapshots,
			SnapshotReader: snapshots,
		}
	})

	Describe("func Load()", func() {
		It("returns zero values when the instance has no historical events", func() {
			nextOffset, snapshotAge, err := loader.Load(
				context.Background(),
				handlerID,
				"<instance>",
				root,
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(nextOffset).To(BeZero())
			Expect(snapshotAge).To(BeZero())
		})
	})
})
