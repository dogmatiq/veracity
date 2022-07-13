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
		ginkgo.It("returns zero offsets when there are no historical events", func() {
			firstOffset, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(firstOffset).To(gomega.BeNumerically("==", 0))
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 0))
		})

		ginkgo.It("returns the next unused offset", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 5))
		})

		ginkgo.It("returns the next unused offset after several writes", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				3,
				allEvents[3:],
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 5))
		})

		ginkgo.It("returns a zero firstOffset when there are historical events", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			firstOffset, _, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(firstOffset).To(gomega.BeNumerically("==", 0))
		})

		ginkgo.It("returns a firstOffset equal to nextOffset when all events are archived", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				3,
				allEvents[3:],
				true, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			firstOffset, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(firstOffset).To(gomega.BeNumerically("==", 5))
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 5))
		})

		ginkgo.It("returns a firstOffset equal to the offset of the first unarchived event", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents[:3],
				true, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				3,
				3,
				allEvents[3:],
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			firstOffset, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(firstOffset).To(gomega.BeNumerically("==", 3))
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 5))
		})
	})

	ginkgo.Describe("func ReadEvents()", func() {
		ginkgo.When("there are no historical events", func() {
			ginkgo.It("returns no events when reading from the next offset (0, in this case)", func() {
				events, more, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(more).To(gomega.BeFalse())
			})

			ginkgo.It("returns no events when reading from a non-existent offset (i.e. any non-zero offset)", func() {
				events, more, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					2,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(more).To(gomega.BeFalse())
			})
		})

		ginkgo.When("there are historical events", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					0,
					allEvents[:3],
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					3,
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

			ginkgo.It("returns the events starting at an offset from the middle of an atomic write", func() {
				expectEvents(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					2,
					allEvents[2:],
				)
			})

			ginkgo.It("returns no events when the offset is the next offset", func() {
				events, more, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					5,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(more).To(gomega.BeFalse())
			})

			ginkgo.It("returns no events when the offset is greater than the next offset", func() {
				events, more, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					6,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(more).To(gomega.BeFalse())
			})
		})

		ginkgo.When("there are archived events", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					0,
					allEvents[:3],
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns an error when the offset refers to an archived event", func() {
				if tc.ArchiveIsSoftDelete {
					ginkgo.Skip("archived events are soft-deleted")
				}

				_, _, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).To(gomega.MatchError("event at offset 0 is archived"))
			})

			ginkgo.It("does not return an error when the offset refers to an event after the last archived event", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					3,
					3,
					allEvents[3:],
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				expectEvents(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					4,
					allEvents[4:],
				)
			})
		})
	})

	ginkgo.Describe("func WriteEvents()", func() {
		ginkgo.When("there are no historical events", func() {
			ginkgo.It("returns an error if firstOffset is not the actual first offset", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					3, // incorrect firstOffset
					0,
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 3 is not the first offset"))
			})

			ginkgo.It("returns an error if nextOffset is not the actual next offset", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					3, // incorrect nextOffset
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 3 is not the next offset"))
			})
		})

		ginkgo.When("there are historical events", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					0,
					allEvents[:3],
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns an error if firstOffset is not the actual first offset", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					4, // incorrect firstOffset
					3,
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 4 is not the first offset"))
			})

			ginkgo.It("returns an error if nextOffset is not the actual next offset", func() {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					3,
					4, // incorrect nextOffset
					allEvents,
					false, // archive
				)
				gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 4 is not the next offset"))
			})
		})

		ginkgo.It("allows archiving without adding more events", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				allEvents,
				false, // archive
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				5,
				nil, // don't add more events
				true,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			firstOffset, nextOffset, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(firstOffset).To(gomega.BeNumerically("==", 5))
			gomega.Expect(nextOffset).To(gomega.BeNumerically("==", 5))
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
				events := allEvents[:i+1]

				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					0,
					0,
					events,
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					uint64(len(events)),
					uint64(len(events)),
					events,
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			for i, inst := range instances {
				firstOffset, nextOffset, err := tc.Reader.ReadBounds(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
				)

				expectedFirstOffset := i + 1
				expectedNextOffset := expectedFirstOffset + i + 1

				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(firstOffset).To(gomega.BeNumerically("==", expectedFirstOffset))
				gomega.Expect(nextOffset).To(gomega.BeNumerically("==", expectedNextOffset))
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
				events := allEvents[:i+1]

				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					0,
					0,
					events,
					true, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					uint64(len(events)),
					uint64(len(events)),
					events,
					false, // archive
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			for i, inst := range instances {
				expectEvents(
					ctx,
					tc.Reader,
					inst.HandlerKey,
					inst.InstanceID,
					uint64(i+1),
					allEvents[:i+1],
				)
			}
		})
	})
}

// expectEvents reads all events from store starting from offset and asserts
// that they are equal to expectedEvents.
func expectEvents(
	ctx context.Context,
	reader aggregate.EventReader,
	hk, id string,
	offset uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, more, err := reader.ReadEvents(
			ctx,
			hk,
			id,
			offset,
		)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if !more {
			break
		}

		offset += uint64(len(events))
	}

	gomega.Expect(producedEvents).To(gomegax.EqualX(expectedEvents))
}
