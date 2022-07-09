package aggregate_test

import (
	"context"
	"errors"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"

	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Loader", func() {
	var (
		handlerID configkit.Identity
		events    *eventReaderStub
		snapshots *memory.AggregateSnapshotStore
		root      *AggregateRoot
		loader    *Loader
	)

	BeforeEach(func() {
		handlerID = configkit.MustNewIdentity("<handler-name>", "<handler>")

		events = &eventReaderStub{
			EventReader: &memory.AggregateEventStore{},
		}

		snapshots = &memory.AggregateSnapshotStore{
			Marshaler: Marshaler,
		}

		root = &AggregateRoot{}

		loader = &Loader{
			EventReader: events,
		}
	})

	Describe("func Load()", func() {
		It("returns an error if event bounds cannot be read", func() {
			events.ReadBoundsFunc = func(
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
			When("there is no snapshot reader", func() {
				XIt("applies all historical events to the root", func() {

				})
			})

			When("there is a snapshot reader", func() {
				BeforeEach(func() {
					loader.SnapshotReader = snapshots
				})

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
				BeforeEach(func() {
					loader.SnapshotWriter = snapshots
				})

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

type eventReaderStub struct {
	EventReader

	ReadBoundsFunc func(
		ctx context.Context,
		hk, id string,
	) (firstOffset, nextOffset uint64, _ error)

	ReadEventsFunc func(
		ctx context.Context,
		hk, id string,
		firstOffset uint64,
	) (events []*envelopespec.Envelope, more bool, _ error)
}

func (s *eventReaderStub) ReadBounds(
	ctx context.Context,
	hk, id string,
) (firstOffset, nextOffset uint64, _ error) {
	if s.ReadBoundsFunc != nil {
		return s.ReadBoundsFunc(ctx, hk, id)
	}

	if s.EventReader != nil {
		return s.EventReader.ReadBounds(ctx, hk, id)
	}

	return 0, 0, nil
}

func (s *eventReaderStub) ReadEvents(
	ctx context.Context,
	hk, id string,
	firstOffset uint64,
) (events []*envelopespec.Envelope, more bool, _ error) {
	if s.ReadEventsFunc != nil {
		return s.ReadEventsFunc(ctx, hk, id, firstOffset)
	}

	if s.EventReader != nil {
		return s.EventReader.ReadEvents(ctx, hk, id, firstOffset)
	}

	return nil, false, nil
}
