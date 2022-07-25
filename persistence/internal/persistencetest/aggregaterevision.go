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

// AggregateRevisionContext encapsulates values used during aggregate revision
// tests.
type AggregateRevisionContext struct {
	Reader                      aggregate.RevisionReader
	Writer                      aggregate.RevisionWriter
	CanReadRevisionsBeforeBegin bool
	AfterEach                   func()
}

// DeclareAggregateRevisionTests declares a function test-suite for persistence
// of aggregate revisions.
func DeclareAggregateRevisionTests(
	new func() AggregateRevisionContext,
) {
	var (
		tc             AggregateRevisionContext
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
		ginkgo.It("returns 0, 0, 0 when there are no revisions", func() {
			begin, committed, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(committed).To(gomega.BeNumerically("==", 0))
			gomega.Expect(end).To(gomega.BeNumerically("==", 0))
		})

		ginkgo.It("returns 0, 0, 1 when there is one uncommitted revision", func() {
			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    0,
					Events: eventA,
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, committed, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(committed).To(gomega.BeNumerically("==", 0))
			gomega.Expect(end).To(gomega.BeNumerically("==", 1))
		})

		ginkgo.It("returns 0, 1, 1 when there is one committed revision", func() {
			rev := aggregate.Revision{
				Begin:  0,
				End:    0,
				Events: eventA,
			}

			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				rev,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				rev,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, committed, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
			gomega.Expect(committed).To(gomega.BeNumerically("==", 1))
			gomega.Expect(end).To(gomega.BeNumerically("==", 1))
		})

		ginkgo.It("allows multiple uncommitted revisions", func() {
			for next := uint64(0); next < 3; next++ {
				err := tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  0,
						End:    next,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				begin, committed, end, err := tc.Reader.ReadBounds(
					ctx,
					"<handler>",
					"<instance>",
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
				gomega.Expect(committed).To(gomega.BeNumerically("==", 0))
				gomega.Expect(end).To(gomega.BeNumerically("==", next+1))
			}

			for next := uint64(0); next < 3; next++ {
				err := tc.Writer.CommitRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  0,
						End:    next,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				begin, committed, end, err := tc.Reader.ReadBounds(
					ctx,
					"<handler>",
					"<instance>",
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", 0))
				gomega.Expect(committed).To(gomega.BeNumerically("==", next+1))
				gomega.Expect(end).To(gomega.BeNumerically("==", 3))
			}
		})

		ginkgo.It("returns n, ?, n when begin == end", func() {
			for next := uint64(0); next < 3; next++ {
				err := tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  next + 1,
						End:    next,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				begin, _, end, err := tc.Reader.ReadBounds(
					ctx,
					"<handler>",
					"<instance>",
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", next+1))
				gomega.Expect(end).To(gomega.BeNumerically("==", next+1))
			}
		})

		ginkgo.It("returns n-1, ?, n when there is a revision written after setting begin == end", func() {
			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  1,
					End:    0,
					Events: eventA,
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  1,
					End:    1,
					Events: eventA,
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, _, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 1))
			gomega.Expect(end).To(gomega.BeNumerically("==", 2))
		})
	})

	ginkgo.Describe("func ReadRevisions()", func() {
		ginkgo.When("there are no revisions", func() {
			ginkgo.It("returns 0 revisions when starting at revision 0", func() {
				revisions, err := tc.Reader.ReadRevisions(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(revisions).To(gomega.BeEmpty())
			})

			ginkgo.It("returns 0 revisions when starting from a non-existent future revision", func() {
				revisions, err := tc.Reader.ReadRevisions(
					ctx,
					"<handler>",
					"<instance>",
					2,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(revisions).To(gomega.BeEmpty())
			})
		})

		ginkgo.When("there are revisions", func() {
			var allRevisions []aggregate.Revision

			ginkgo.BeforeEach(func() {
				for next := uint64(0); next < 3; next++ {
					events := []*envelopespec.Envelope{
						veracityfixtures.NewEnvelope(
							"<event>",
							dogmafixtures.MessageE{
								Value: fmt.Sprintf("<event-%d-a>", next),
							},
						),
						veracityfixtures.NewEnvelope(
							"<event>",
							dogmafixtures.MessageE{
								Value: fmt.Sprintf("<event-%d-b>", next),
							},
						),
					}

					rev := aggregate.Revision{
						Begin:  0,
						End:    next,
						Events: events,
					}

					err := tc.Writer.PrepareRevision(
						ctx,
						"<handler>",
						"<instance>",
						rev,
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

					allRevisions = append(allRevisions, rev)
				}
			})

			ginkgo.It("returns the revisions in the order they were prepared", func() {
				expectRevisions(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					0,
					allRevisions,
				)
			})

			ginkgo.It("returns 0 revisions when starting from the next revision", func() {
				revisions, err := tc.Reader.ReadRevisions(
					ctx,
					"<handler>",
					"<instance>",
					3,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(revisions).To(gomega.BeEmpty())
			})

			ginkgo.It("returns 0 revisions when starting from a non-existent future revision after the next revision", func() {
				revisions, err := tc.Reader.ReadRevisions(
					ctx,
					"<handler>",
					"<instance>",
					6,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(revisions).To(gomega.BeEmpty())
			})
		})

		ginkgo.When("the begin revision has been advanced beyond zero", func() {
			ginkgo.BeforeEach(func() {
				err := tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin: 0,
						End:   0,
						Events: []*envelopespec.Envelope{
							veracityfixtures.NewEnvelope(
								"<archived-event>",
								dogmafixtures.MessageX1,
							),
						},
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  1,
						End:    1,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				err = tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  1,
						End:    2,
						Events: eventB,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("returns an error when reading from before the new begin revision", func() {
				if tc.CanReadRevisionsBeforeBegin {
					ginkgo.Skip("this implementation allows reading revisions before the begin revision")
				}

				_, err := tc.Reader.ReadRevisions(
					ctx,
					"<handler>",
					"<instance>",
					0,
				)
				gomega.Expect(err).To(
					gomega.MatchError(
						"revision 0 is archived",
					),
				)
			})

			ginkgo.It("allows reading from the new begin revision", func() {
				expectRevisions(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					1,
					[]aggregate.Revision{
						{
							Begin:  1,
							End:    1,
							Events: eventA,
						},
						{
							Begin:  1,
							End:    2,
							Events: eventB,
						},
					},
				)
			})

			ginkgo.It("allows reading after the new begin revision", func() {
				expectRevisions(
					ctx,
					tc.Reader,
					"<handler>",
					"<instance>",
					2,
					[]aggregate.Revision{
						{
							Begin:  1,
							End:    2,
							Events: eventB,
						},
					},
				)
			})
		})
	})

	ginkgo.Describe("func PrepareRevision()", func() {
		ginkgo.It("returns an error if the given revision already exists", func() {
			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    0,
					Events: eventA,
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    0, // incorrect end revision
					Events: eventB,
				},
			)
			gomega.Expect(err).To(
				gomega.MatchError(
					"optimistic concurrency conflict, 0 is not the next revision",
				),
			)
		})

		ginkgo.It("allows increasing begin without writing any events", func() {
			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  1,
					End:    0,
					Events: eventA,
				},
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			begin, committed, end, err := tc.Reader.ReadBounds(
				ctx,
				"<handler>",
				"<instance>",
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(begin).To(gomega.BeNumerically("==", 1))
			gomega.Expect(committed).To(gomega.BeNumerically("==", 0))
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
				var next uint64

				for j := 0; j < i; j++ {
					err := tc.Writer.PrepareRevision(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						aggregate.Revision{
							Begin:  0,
							End:    next,
							Events: eventA,
						},
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
					next++
				}

				begin := next + 1
				err := tc.Writer.PrepareRevision(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					aggregate.Revision{
						Begin: begin,
						End:   next,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				next++

				err = tc.Writer.PrepareRevision(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					aggregate.Revision{
						Begin:  begin,
						End:    next,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			for i, inst := range instances {
				expectedBegin := i + 1
				expectedEnd := expectedBegin + 1

				begin, _, end, err := tc.Reader.ReadBounds(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", expectedBegin))
				gomega.Expect(end).To(gomega.BeNumerically("==", expectedEnd))
			}
		})

		ginkgo.It("stores separate revisions for each combination of handler key and instance ID", func() {
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

			var revisions []aggregate.Revision

			for i, inst := range instances {
				env := veracityfixtures.NewEnvelope(
					"<event>",
					dogmafixtures.MessageE{
						Value: fmt.Sprintf("<event-%d>", i),
					},
				)

				rev := aggregate.Revision{
					Begin:  0,
					End:    0,
					Events: []*envelopespec.Envelope{env},
				}

				err := tc.Writer.PrepareRevision(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
					rev,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				revisions = append(revisions, rev)
			}

			for i, inst := range instances {
				expectRevisions(
					ctx,
					tc.Reader,
					inst.HandlerKey,
					inst.InstanceID,
					0,
					revisions[i:i+1],
				)
			}
		})
	})

	ginkgo.Describe("func CommitRevision()", func() {
		ginkgo.It("returns an error if the revision does not exist", func() {
			err := tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    0,
					Events: eventA,
				},
			)
			gomega.Expect(err).To(gomega.MatchError("revision 0 does not exist"))

			err = tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    1,
					Events: eventA,
				},
			)
			gomega.Expect(err).To(gomega.MatchError("revision 1 does not exist"))
		})

		ginkgo.It("returns an error if the revision is already committed", func() {
			rev := aggregate.Revision{
				Begin:  0,
				End:    0,
				Events: eventA,
			}

			err := tc.Writer.PrepareRevision(
				ctx,
				"<handler>",
				"<instance>",
				rev,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				rev,
			)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				rev,
			)
			gomega.Expect(err).To(gomega.MatchError("revision 0 is already committed"))
		})

		ginkgo.It("returns an error if there are earlier uncommitted revisions", func() {
			for next := uint64(0); next < 3; next++ {
				err := tc.Writer.PrepareRevision(
					ctx,
					"<handler>",
					"<instance>",
					aggregate.Revision{
						Begin:  0,
						End:    next,
						Events: eventA,
					},
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}

			err := tc.Writer.CommitRevision(
				ctx,
				"<handler>",
				"<instance>",
				aggregate.Revision{
					Begin:  0,
					End:    1,
					Events: eventA,
				},
			)
			gomega.Expect(err).To(gomega.MatchError("cannot commit revision 1, the previous revision is not committed"))
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
				for next := uint64(0); next < uint64(len(instances)); next++ {
					err := tc.Writer.PrepareRevision(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						aggregate.Revision{
							Begin:  0,
							End:    next,
							Events: eventA,
						},
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				}

				for next := uint64(0); next < uint64(i); next++ {
					err := tc.Writer.CommitRevision(
						ctx,
						inst.HandlerKey,
						inst.InstanceID,
						aggregate.Revision{
							Begin:  0,
							End:    next,
							Events: eventA,
						},
					)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				}
			}

			for i, inst := range instances {
				expectedBegin := 0
				expectedCommitted := i
				expectedEnd := len(instances)

				begin, committed, end, err := tc.Reader.ReadBounds(
					ctx,
					inst.HandlerKey,
					inst.InstanceID,
				)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				gomega.Expect(begin).To(gomega.BeNumerically("==", expectedBegin))
				gomega.Expect(committed).To(gomega.BeNumerically("==", expectedCommitted))
				gomega.Expect(end).To(gomega.BeNumerically("==", expectedEnd))
			}
		})
	})
}

// expectEvents reads all revisions starting from begin and asserts that they
// are equal to expected.
func expectRevisions(
	ctx context.Context,
	reader aggregate.RevisionReader,
	hk, id string,
	begin uint64,
	expected []aggregate.Revision,
) {
	var actual []aggregate.Revision

	for {
		revisions, err := reader.ReadRevisions(
			ctx,
			hk,
			id,
			begin,
		)
		gomega.ExpectWithOffset(1, err).ShouldNot(gomega.HaveOccurred())

		if len(revisions) == 0 {
			break
		}

		actual = append(actual, revisions...)
		begin += uint64(len(revisions))
	}

	if len(actual) == 0 && len(expected) == 0 {
		return
	}

	gomega.ExpectWithOffset(1, actual).To(gomegax.EqualX(expected))
}
