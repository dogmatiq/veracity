package persistencetest

import (
	"context"
	"fmt"
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
	Reader                      aggregate.EventReader
	Writer                      aggregate.EventWriter
	CanReadRevisionsBeforeBegin bool
	AfterEach                   func()
}

// DeclareAggregateEventTests declares a function test-suite for persistence of
// aggregate events.
func DeclareAggregateEventTests(
	new func() AggregateEventContext,
) {
	var (
		tc             AggregateEventContext
		ctx            context.Context
		eventA, eventB []*envelopespec.Envelope
	)

	ginkgo.BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		ginkgo.DeferCleanup(cancel)

		eventA = []*envelopespec.Envelope{
			veracityfixtures.NewEnvelope("<event-0>", dogmafixtures.MessageA1),
		}

		eventB = []*envelopespec.Envelope{
			veracityfixtures.NewEnvelope("<event-1>", dogmafixtures.MessageB1),
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

		ginkgo.It("returns [0, n) after n writes", func() {
			for rev := uint64(0); rev < 3; rev++ {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
					rev,
					eventA,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				begin, end, err := tc.Reader.ReadBounds(
					ctx,
					"<handler>",
					"<instance>",
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
				gomega.Expect(end).To(gomega.BeNumerically("==", rev+1))
			}
		})

		ginkgo.It("returns [n, n) when begin == end", func() {
			for rev := uint64(0); rev < 3; rev++ {
				err := tc.Writer.WriteEvents(
					ctx,
					"<handler>",
					"<instance>",
					rev+1,
					rev,
					eventA,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				begin, end, err := tc.Reader.ReadBounds(
					ctx,
					"<handler>",
					"<instance>",
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", rev+1))
				gomega.Expect(end).To(gomega.BeNumerically("==", rev+1))
			}
		})

		ginkgo.It("returns [n-1, n) when there is a revision written after setting begin == end", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				0,
				eventA,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				1,
				eventA,
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
		ginkgo.When("there are no historical events", func() {
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

		ginkgo.When("there are no historical events, but there are prior revisions", func() {
			ginkgo.BeforeEach(func() {
				for rev := uint64(0); rev < 3; rev++ {
					err := tc.Writer.WriteEvents(
						ctx,
						"<handler>",
						"<instance>",
						0,
						rev,
						nil, // no events
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				}
			})

			ginkgo.It("returns 0 events when starting at revision 0", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("==", 3))
			})

			ginkgo.It("returns 0 events when starting from a non-zero revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					1,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("==", 3))
			})

			ginkgo.It("returns 0 events when starting from a non-existent future revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					3,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("==", 3))
			})
		})

		ginkgo.When("there are historical events", func() {
			var historicalEvents []*envelopespec.Envelope

			ginkgo.BeforeEach(func() {
				for rev := uint64(0); rev < 3; rev++ {
					events := []*envelopespec.Envelope{
						veracityfixtures.NewEnvelope(
							"<event>",
							dogmafixtures.MessageE{
								Value: fmt.Sprintf("<event-%d-a>", rev),
							},
						),
						veracityfixtures.NewEnvelope(
							"<event>",
							dogmafixtures.MessageE{
								Value: fmt.Sprintf("<event-%d-b>", rev),
							},
						),
					}

					err := tc.Writer.WriteEvents(
						ctx,
						"<handler>",
						"<instance>",
						0,
						rev,
						events,
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

					historicalEvents = append(historicalEvents, events...)
				}
			})

			ginkgo.It("returns the events in the order they were written", func() {
				expectEvents(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					0,
					historicalEvents,
				)
			})

			ginkgo.It("returns 0 events when starting from the next revision", func() {
				events, end, err := tc.Reader.ReadEvents(
					ctx,
					"<handler>",
					"<instance>",
					3,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(events).To(gomega.BeEmpty())
				gomega.Expect(end).To(gomega.BeNumerically("==", 3))
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
	})

	ginkgo.When("the begin offset has been advanced beyond zero", func() {
		ginkgo.BeforeEach(func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				[]*envelopespec.Envelope{
					veracityfixtures.NewEnvelope(
						"<archived-event>",
						dogmafixtures.MessageX1,
					),
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				1,
				eventA,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				2,
				eventB,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("returns an error when reading from before the new begin revision", func() {
			if tc.CanReadRevisionsBeforeBegin {
				ginkgo.Skip("this implementation allows reading revisions before the begin revision")
			}

			_, _, err := tc.Reader.ReadEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
			)
			gomega.Expect(err).To(gomega.MatchError("revision 0 is archived"))
		})

		ginkgo.It("allows reading from the new begin revision", func() {
			expectEvents(
				ctx,
				tc.Reader,
				"<handler>",
				"<instance>",
				1,
				append(eventA, eventB...),
			)
		})

		ginkgo.It("allows reading after the new begin revision", func() {
			expectEvents(
				ctx,
				tc.Reader,
				"<handler>",
				"<instance>",
				2,
				eventB,
			)
		})
	})

	ginkgo.Describe("func WriteEvents()", func() {
		ginkgo.It("returns an error if the given revision already exists", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0,
				eventA,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				0,
				0, // incorrect end revision
				eventB,
			)
			gomega.Expect(err).To(gomega.MatchError("optimistic concurrency conflict, 0 is not the next revision"))
		})

		ginkgo.It("allows increasing begin without writing any events", func() {
			err := tc.Writer.WriteEvents(
				ctx,
				"<handler>",
				"<instance>",
				1,
				0,
				eventA,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 1))
			gomega.Expect(end).To(gomega.BeNumerically("==", 1))
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
				var rev uint64

				for j := 0; j < i; j++ {
					err := tc.Writer.WriteEvents(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						0,
						rev,
						eventA,
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					rev++
				}

				begin := rev + 1
				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					begin,
					rev,
					nil,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				rev++

				err = tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					begin,
					rev,
					eventA,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			for i, inst := range instances {
				expectedBegin := i + 1
				expectedEnd := expectedBegin + 1

				begin, end, err := tc.Reader.ReadBounds(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
				)
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

			var envelopes []*envelopespec.Envelope

			for i, inst := range instances {
				env := veracityfixtures.NewEnvelope(
					"<event>",
					dogmafixtures.MessageE{
						Value: fmt.Sprintf("<event-%d>", i),
					},
				)
				err := tc.Writer.WriteEvents(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					0,
					0,
					[]*envelopespec.Envelope{env},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				envelopes = append(envelopes, env)
			}

			for i, inst := range instances {
				expectEvents(
					ctx,
					tc.Reader,
					inst.HandlerKey,
					inst.InstanceID,
					0,
					envelopes[i:i+1],
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

		if begin == end {
			break
		}

		begin = end
	}

	if len(producedEvents) == 0 && len(expectedEvents) == 0 {
		return
	}

	gomega.Expect(producedEvents).To(gomegax.EqualX(expectedEvents))
}
