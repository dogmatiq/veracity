package aggregate_test

import (
	"context"
	"errors"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Loader", func() {
	var (
		handlerID      configkit.Identity
		eventStore     *memory.AggregateEventStore
		eventReader    *eventReaderStub
		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		root           *AggregateRoot
		loader         *Loader
	)

	BeforeEach(func() {
		handlerID = configkit.MustNewIdentity("<handler-name>", "<handler>")

		eventStore = &memory.AggregateEventStore{}

		eventReader = &eventReaderStub{
			EventReader: eventStore,
		}

		snapshotStore = &memory.AggregateSnapshotStore{
			Marshaler: Marshaler,
		}

		snapshotReader = &snapshotReaderStub{
			SnapshotReader: snapshotStore,
		}

		root = &AggregateRoot{}

		loader = &Loader{
			EventReader: eventReader,
			Marshaler:   Marshaler,
		}
	})

	Describe("func Load()", func() {
		It("returns an error if event bounds cannot be read", func() {
			eventReader.ReadBoundsFunc = func(
				ctx context.Context,
				hk, id string,
			) (firstOffset uint64, nextOffset uint64, _ error) {
				return 0, 0, errors.New("<error>")
			}

			_, _, err := loader.Load(
				context.Background(),
				handlerID,
				"<instance>",
				root,
			)
			Expect(err).To(
				MatchError(
					`aggregate root <handler-name>[<instance>] cannot be loaded: unable to read event offset bounds: <error>`,
				),
			)
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
			BeforeEach(func() {
				eventStore.WriteEvents(
					context.Background(),
					handlerID.Key,
					"<instance>",
					0,
					[]*envelopespec.Envelope{
						NewEnvelope("<event-0>", MessageA1),
						NewEnvelope("<event-1>", MessageB1),
						NewEnvelope("<event-2>", MessageC1),
					},
					false,
				)
			})

			When("there is no snapshot reader", func() {
				It("applies all historical events to the root", func() {
					nextOffset, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(nextOffset).To(BeNumerically("==", 3))
					Expect(snapshotAge).To(BeNumerically("==", 3))
					Expect(root.AppliedEvents).To(Equal(
						[]dogma.Message{
							MessageA1,
							MessageB1,
							MessageC1,
						},
					))
				})
			})

			When("there is a snapshot reader", func() {
				BeforeEach(func() {
					loader.SnapshotReader = snapshotReader
				})

				It("returns an error if the context is cancelled while reading the snapshot", func() {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					snapshotReader.ReadSnapshotFunc = func(
						ctx context.Context,
						hk, id string,
						r dogma.AggregateRoot,
						minOffset uint64,
					) (snapshotOffset uint64, ok bool, _ error) {
						cancel()
						return 0, false, ctx.Err()
					}

					_, _, err := loader.Load(
						ctx,
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).To(
						MatchError(
							`aggregate root <handler-name>[<instance>] cannot be loaded: unable to read snapshot: context canceled`,
						),
					)
				})

				It("applies all historical events if the snapshot reader fails", func() {
					snapshotReader.ReadSnapshotFunc = func(
						ctx context.Context,
						hk, id string,
						r dogma.AggregateRoot,
						minOffset uint64,
					) (snapshotOffset uint64, ok bool, _ error) {
						return 0, false, errors.New("<error>")
					}

					nextOffset, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(nextOffset).To(BeNumerically("==", 3))
					Expect(snapshotAge).To(BeNumerically("==", 3))
					Expect(root.AppliedEvents).To(Equal(
						[]dogma.Message{
							MessageA1,
							MessageB1,
							MessageC1,
						},
					))
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
				BeforeEach(func() {
					loader.SnapshotWriter = snapshotStore
				})

				XIt("writes a snapshot if the event reader fails after some events are already applied to the root", func() {

				})

				XIt("writes a snapshot if the unmarshaling fails after some events are already applied to the root", func() {

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
