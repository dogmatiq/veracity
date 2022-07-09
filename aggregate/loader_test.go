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

		root = &AggregateRoot{}

		loader = &Loader{
			EventReader:    events,
			SnapshotWriter: snapshots,
			SnapshotReader: snapshots,
		}
	})

	Describe("func Load()", func() {
		XIt("returns an error if event bounds cannot be read", func() {

		})

		When("the instance has no historical events", func() {
			It("does not modify the root", func() {
				nextOffset, snapshotAge, err := loader.Load(
					context.Background(),
					handlerID,
					"<instance>",
					root,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(nextOffset).To(BeZero())
				Expect(snapshotAge).To(BeZero())
				Expect(root.AppliedEvents).To(BeEmpty())
			})
		})

		When("the instance has historical events", func() {
			When("there is no snapshot reader", func() {
				XIt("applies all historical events to the root", func() {

				})
			})

			When("there is a snapshot reader", func() {
				XIt("returns an error if the context is cancelled while reading the snapshot", func() {

				})

				XIt("applies all historical events if the snapshot reader fails", func() {

				})

				When("there is a snapshot available", func() {
					When("the snapshot is up-to-date", func() {
						XIt("does not apply any events", func() {

						})
					})

					When("the snapshot is out-of-date", func() {
						XIt("applies only those events that occurred after the snapshot", func() {

						})
					})
				})
			})

			When("there is a snapshot writer", func() {
				XIt("writes a snapshot if the event reader fails after some events are already applied to the root", func() {

				})

				XIt("does not write a snapshot if the event reader fails immediately", func() {

				})

				XIt("does not write a snapshot if all events are applied to the root", func() {

				})
			})

			When("the instance has been destroyed", func() {
				When("the instance has been recreated", func() {
					XIt("applies only those events that occurred after destruction", func() {

					})

					When("there is a snapshot available", func() {
						XIt("applies only those events that occurred after the snapshot", func() {

						})

						XIt("ignores snapshots taken before destruction", func() {

						})
					})
				})

				When("the instance has not been recreated", func() {
					XIt("does not apply any events", func() {

					})
				})
			})
		})
	})
})
