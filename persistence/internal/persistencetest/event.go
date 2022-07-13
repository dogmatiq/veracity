package persistencetest

import (
	"context"
	"time"

	dogmafixtures "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/aggregate"
	veracityfixtures "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/jmalloc/gomegax"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// AggregateEventContext encapsulates values used during event tests.
type AggregateEventContext struct {
	Reader              aggregate.EventReader
	Writer              aggregate.EventWriter
	ArchiveIsSoftDelete bool
	AfterEach           func()
}

// DeclareAggregateEventTests declares a function test-suite for persistence of
// aggregate events.
func DeclareAggregateEventTests(
	new func() AggregateEventContext,
) {
	var (
		tc        AggregateEventContext
		ctx       context.Context
		allEvents []*envelopespec.Envelope
	)

	ginkgo.BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		ginkgo.DeferCleanup(cancel)

		allEvents = []*envelopespec.Envelope{
			veracityfixtures.NewEnvelope("<event-0>", dogmafixtures.MessageA1),
			veracityfixtures.NewEnvelope("<event-1>", dogmafixtures.MessageB1),
			veracityfixtures.NewEnvelope("<event-2>", dogmafixtures.MessageC1),
			veracityfixtures.NewEnvelope("<event-4>", dogmafixtures.MessageD1),
			veracityfixtures.NewEnvelope("<event-5>", dogmafixtures.MessageE1),
		}

		tc = new()
		if tc.AfterEach != nil {
			ginkgo.DeferCleanup(tc.AfterEach)
		}
	})

	ginkgo.Describe("func ReadBounds()", func() {
		ginkgo.It("returns [0, 0) when there have not been any writes", func() {
			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(end).To(gomega.BeNumerically("==", 0))
		})

		ginkgo.It("returns [0, 1) after a single write", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(end).To(gomega.BeNumerically("==", 1))
		})

		ginkgo.It("returns [0, 2) after two writes", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(end).To(gomega.BeNumerically("==", 2))
		})

		ginkgo.It("returns [n, n) when all events are archived", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				allEvents,
				true, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 2))
			gomega.Expect(end).To(gomega.BeNumerically("==", 2))
		})

		ginkgo.It("returns [n-1, n) when there is a revision written after archiving", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				allEvents,
				true, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 1))
			gomega.Expect(end).To(gomega.BeNumerically("==", 2))
		})
	})

	ginkgo.Describe("func ReadEvents()", func() {
		ginkgo.When("no writes have occurred", func() {
			ginkgo.It("returns 0 events when starting at revision 0", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("==", 0))
			})

			ginkgo.It("returns 0 events when starting from a non-existent future revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					2,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("<=", 2))
			})
		})

		ginkgo.When("writes have occurred", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					allEvents[:3],
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					1,
					allEvents[3:],
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns the events in the order they were written", func() {
				expectEvents(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					0,
					allEvents,
				)
			})

			ginkgo.It("returns 0 events when starting from the next revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					2,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("<=", 2))
			})

			ginkgo.It("returns 0 events when starting from a non-existent future revision after the next revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					6,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("<=", 6))
			})
		})

		ginkgo.When("there are archived events", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					allEvents[:3],
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns an error when the revision refers to an archived revision", func() {
				if tc.ArchiveIsSoftDelete {
					ginkgo.Skip("archived events are soft-deleted")
				}

				_, _, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).To(gomega.MatchError("revision 0 is archived"))
			})

			ginkgo.It("does not return an error when the revision is after the last archived revision", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					1,
					allEvents[3:],
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				expectEvents(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					1,
					allEvents[3:],
				)
			})
		})
	})

	ginkgo.Describe("func WriteEvents()", func() {
		ginkgo.When("no writes have occurred", func() {
			ginkgo.It("returns an error if the given revision is not the (exclusive) end revision", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					3, // incorrect end revision
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 3 is not the next revision"))
			})
		})

		ginkgo.When("writes have occurred", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					allEvents,
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns an error if the given revision is not the (exclusive) end revision", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					3, // incorrect end revision
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 3 is not the next revision"))
			})
		})

		ginkgo.It("allows archiving without writing events", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				nil, // don't add more events
				true,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 2))
			gomega.Expect(end).To(gomega.BeNumerically("==", 2))
		})

		ginkgo.It("stores separate bounds for each combination of handler key and instance ID", func() {
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
				events := allEvents[:i]

				var rev uint64
				for _, ev := range events {
					err := tc.Writer.WriteEvents(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						rev,
						[]*envelopespec.Envelope{ev},
						false, // archive
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					rev++
				}

				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					rev,
					[]*envelopespec.Envelope{},
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				rev++

				for _, ev := range events {
					err = tc.Writer.WriteEvents(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						rev,
						[]*envelopespec.Envelope{ev},
						false, // archive
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					rev++
				}
			}

			for i, inst := range instances {
				begin, end, err := tc.Reader.ReadBounds(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
				)

				expectedBegin := i + 1
				expectedEnd := (i * 2) + 1

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", expectedBegin))
				gomega.Expect(end).To(gomega.BeNumerically("==", expectedEnd))
			}
		})

		ginkgo.It("stores separate events for each combination of handler key and instance ID", func() {
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
				events := allEvents[:i]

				var rev uint64
				for _, ev := range events {
					err := tc.Writer.WriteEvents(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						rev,
						[]*envelopespec.Envelope{ev},
						false, // archive
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					rev++
				}

				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					rev,
					[]*envelopespec.Envelope{},
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				rev++

				for _, ev := range events {
					err = tc.Writer.WriteEvents(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						rev,
						[]*envelopespec.Envelope{ev},
						false, // archive
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					rev++
				}
			}

			for i, inst := range instances {
				expectEvents(
					ctx,
					tc.Reader,
					inst.HandlerKey,
					inst.InstanceID,
					uint64(i+1),
					allEvents[:i],
				)
			}
		})
	})
}

// expectEvents reads all events from store starting from begin and asserts
// that they are equal to expectedEvents.
func expectEvents(
	ctx context.Context,
	reader aggregate.EventReader,
	hk, id string,
	begin uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, end, err := reader.ReadEvents(
			ctx,
			hk,
			id,
			begin,
		)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if begin >= end {
			break
		}

		begin = end
	}

	if len(producedEvents) == 0 && len(expectedEvents) == 0 {
		return
	}

	gomega.Expect(producedEvents).To(gomegax.EqualX(expectedEvents))
}
