package aggregate_test

import (
	"context"
	"errors"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("type RevisionLoader", func() {
	var (
		revisionStore  *memory.AggregateRevisionStore
		revisionReader *revisionReaderStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		handlerID configkit.Identity
		root      *AggregateRoot
		loader    *RevisionLoader
	)

	BeforeEach(func() {
		handlerID = configkit.MustNewIdentity("<handler-name>", "<handler>")

		revisionStore = &memory.AggregateRevisionStore{}

		revisionReader = &revisionReaderStub{
			RevisionReader: revisionStore,
		}

		snapshotStore = &memory.AggregateSnapshotStore{
			Marshaler: Marshaler,
		}

		snapshotReader = &snapshotReaderStub{
			SnapshotReader: snapshotStore,
		}

		snapshotWriter = &snapshotWriterStub{
			SnapshotWriter: snapshotStore,
		}

		root = &AggregateRoot{}

		logger, err := zap.NewDevelopment(
			zap.AddStacktrace(zap.PanicLevel + 1),
		)
		Expect(err).ShouldNot(HaveOccurred())

		loader = &RevisionLoader{
			RevisionReader: revisionReader,
			Marshaler:      Marshaler,
			Logger:         logger,
		}
	})

	Describe("func Load()", func() {
		It("returns an error if bounds cannot be read", func() {
			revisionReader.ReadBoundsFunc = func(
				ctx context.Context,
				hk, id string,
			) (uint64, uint64, uint64, error) {
				return 0, 0, 0, errors.New("<error>")
			}

			_, _, _, err := loader.Load(
				context.Background(),
				handlerID,
				"<instance>",
				root,
			)
			Expect(err).To(
				MatchError(
					"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revision bounds: <error>",
				),
			)
		})

		When("the instance has no revisions", func() {
			It("does not modify the root", func() {
				begin, end, snapshotAge, err := loader.Load(
					context.Background(),
					handlerID,
					"<instance>",
					root,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(begin).To(BeNumerically("==", 0))
				Expect(end).To(BeNumerically("==", 0))
				Expect(snapshotAge).To(BeNumerically("==", 0))
				Expect(root.AppliedEvents).To(BeEmpty())
			})
		})

		When("the instance has revisions", func() {
			BeforeEach(func() {
				err := revisionStore.PrepareRevision(
					context.Background(),
					handlerID.Key,
					"<instance>",
					aggregate.Revision{
						Begin: 0,
						End:   0,
						Events: []*envelopespec.Envelope{
							NewEnvelope("<event-0>", MessageA1),
							NewEnvelope("<event-1>", MessageB1),
							NewEnvelope("<event-2>", MessageC1),
						},
					},
				)
				Expect(err).ShouldNot(HaveOccurred())
			})

			When("there is no snapshot reader", func() {
				It("applies all historical events to the root", func() {
					begin, end, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(begin).To(BeNumerically("==", 0))
					Expect(end).To(BeNumerically("==", 1))
					Expect(snapshotAge).To(BeNumerically("==", 1))
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
						minRev uint64,
					) (uint64, bool, error) {
						cancel()
						return 0, false, ctx.Err()
					}

					_, _, _, err := loader.Load(
						ctx,
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).To(
						MatchError(
							"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read snapshot: context canceled",
						),
					)
				})

				It("applies all historical events if the snapshot reader fails", func() {
					snapshotReader.ReadSnapshotFunc = func(
						ctx context.Context,
						hk, id string,
						r dogma.AggregateRoot,
						minRev uint64,
					) (uint64, bool, error) {
						return 0, false, errors.New("<error>")
					}

					begin, end, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(begin).To(BeNumerically("==", 0))
					Expect(end).To(BeNumerically("==", 1))
					Expect(snapshotAge).To(BeNumerically("==", 1))
					Expect(root.AppliedEvents).To(Equal(
						[]dogma.Message{
							MessageA1,
							MessageB1,
							MessageC1,
						},
					))
				})

				When("there is a snapshot available", func() {
					BeforeEach(func() {
						err := snapshotStore.WriteSnapshot(
							context.Background(),
							handlerID.Key,
							"<instance>",
							&AggregateRoot{
								AppliedEvents: []dogma.Message{
									"<snapshot>",
								},
							},
							0,
						)
						Expect(err).ShouldNot(HaveOccurred())
					})

					When("the snapshot is up-to-date", func() {
						It("does not apply any events", func() {
							begin, end, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(begin).To(BeNumerically("==", 0))
							Expect(end).To(BeNumerically("==", 1))
							Expect(snapshotAge).To(BeNumerically("==", 0))
							Expect(root.AppliedEvents).To(Equal(
								[]dogma.Message{
									"<snapshot>",
								},
							))
						})
					})

					When("the snapshot is out-of-date", func() {
						BeforeEach(func() {
							err := revisionStore.PrepareRevision(
								context.Background(),
								handlerID.Key,
								"<instance>",
								aggregate.Revision{
									Begin: 0,
									End:   1,
									Events: []*envelopespec.Envelope{
										NewEnvelope("<event-3>", MessageD1),
									},
								},
							)
							Expect(err).ShouldNot(HaveOccurred())
						})

						It("applies only those events that occurred after the snapshot", func() {
							begin, end, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(begin).To(BeNumerically("==", 0))
							Expect(end).To(BeNumerically("==", 2))
							Expect(snapshotAge).To(BeNumerically("==", 1))
							Expect(root.AppliedEvents).To(Equal(
								[]dogma.Message{
									"<snapshot>",
									MessageD1,
								},
							))
						})
					})
				})
			})

			When("there is no snapshot writer", func() {
				When("a subset of historical events are applied", func() {
					It("does not attempt to write a snapshot", func() {
						revisionReader.ReadRevisionsFunc = func(
							ctx context.Context,
							hk, id string,
							begin uint64,
						) ([]Revision, error) {
							revisions, _ := revisionStore.ReadRevisions(ctx, hk, id, begin)
							if len(revisions) > 0 {
								return revisions, nil
							}

							return nil, errors.New("<error>")
						}

						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).To(
							MatchError(
								"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revisions: <error>",
							),
						)
					})
				})
			})

			When("there is a snapshot writer", func() {
				BeforeEach(func() {
					loader.SnapshotWriter = snapshotWriter
				})

				When("all historical events are applied", func() {
					It("does not write a snapshot", func() {
						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).ShouldNot(HaveOccurred())

						_, ok, err := snapshotStore.ReadSnapshot(
							context.Background(),
							handlerID.Key,
							"<instance>",
							root,
							0,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ok).To(BeFalse())
					})
				})

				When("no historical events are applied", func() {
					It("does not write a snapshot", func() {
						revisionReader.ReadRevisionsFunc = func(
							ctx context.Context,
							hk, id string,
							begin uint64,
						) ([]Revision, error) {
							return nil, errors.New("<error>")
						}

						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).To(
							MatchError(
								"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revisions: <error>",
							),
						)

						_, ok, err := snapshotStore.ReadSnapshot(
							context.Background(),
							handlerID.Key,
							"<instance>",
							root,
							0,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ok).To(BeFalse())
					})
				})

				When("a subset of historical events are applied", func() {
					It("writes a snapshot if the revision reader fails", func() {
						revisionReader.ReadRevisionsFunc = func(
							ctx context.Context,
							hk, id string,
							begin uint64,
						) ([]Revision, error) {
							revisions, _ := revisionStore.ReadRevisions(ctx, hk, id, begin)
							if len(revisions) > 0 {
								return revisions, nil
							}

							return nil, errors.New("<error>")
						}

						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).To(
							MatchError(
								"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revisions: <error>",
							),
						)

						snapshotRev, ok, err := snapshotStore.ReadSnapshot(
							context.Background(),
							handlerID.Key,
							"<instance>",
							root,
							0,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ok).To(BeTrue())
						Expect(snapshotRev).To(BeNumerically("==", 0))
						Expect(root.AppliedEvents).To(Equal(
							[]dogma.Message{
								map[string]any{"Value": "A1"},
								map[string]any{"Value": "B1"},
								map[string]any{"Value": "C1"},
							},
						))
					})

					It("writes a snapshot if unmarshaling an envelope fails", func() {
						done := false
						revisionReader.ReadRevisionsFunc = func(
							ctx context.Context,
							hk, id string,
							begin uint64,
						) ([]Revision, error) {
							revisions, _ := revisionStore.ReadRevisions(ctx, hk, id, begin)
							if len(revisions) > 0 {
								return revisions, nil
							}

							if done {
								return nil, nil
							}

							done = true

							return []Revision{
								{
									Begin: 0,
									End:   begin,
									Events: []*envelopespec.Envelope{
										{ /*empty envelope*/ },
									},
								},
							}, nil
						}

						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).To(
							MatchError(
								"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revisions: mime: no media type",
							),
						)

						snapshotRev, ok, err := snapshotStore.ReadSnapshot(
							context.Background(),
							handlerID.Key,
							"<instance>",
							root,
							0,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(ok).To(BeTrue())
						Expect(snapshotRev).To(BeNumerically("==", 0))
						Expect(root.AppliedEvents).To(Equal(
							[]dogma.Message{
								map[string]any{"Value": "A1"},
								map[string]any{"Value": "B1"},
								map[string]any{"Value": "C1"},
							},
						))
					})

					It("does not return a snapshot-related error if the snapshot cannot be written", func() {
						revisionReader.ReadRevisionsFunc = func(
							ctx context.Context,
							hk, id string,
							begin uint64,
						) ([]Revision, error) {
							revisions, _ := revisionStore.ReadRevisions(ctx, hk, id, begin)
							if len(revisions) > 0 {
								return revisions, nil
							}

							return nil, errors.New("<causal error>")
						}

						snapshotWriter.WriteSnapshotFunc = func(
							ctx context.Context,
							hk, id string,
							r dogma.AggregateRoot,
							rev uint64,
						) error {
							return errors.New("<snapshot error>")
						}

						_, _, _, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).To(
							MatchError(
								"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revisions: <causal error>",
							),
						)
					})
				})
			})

			When("the instance has been destroyed", func() {
				BeforeEach(func() {
					err := revisionStore.PrepareRevision(
						context.Background(),
						handlerID.Key,
						"<instance>",
						Revision{
							Begin: 2,
							End:   1,
						},
					)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("does not apply any events", func() {
					begin, end, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(begin).To(BeNumerically("==", 2))
					Expect(end).To(BeNumerically("==", 2))
					Expect(snapshotAge).To(BeNumerically("==", 0))
					Expect(root.AppliedEvents).To(BeEmpty())
				})

				When("the instance has been recreated", func() {
					BeforeEach(func() {
						err := revisionStore.PrepareRevision(
							context.Background(),
							handlerID.Key,
							"<instance>",
							Revision{
								Begin: 2,
								End:   2,
								Events: []*envelopespec.Envelope{
									NewEnvelope("<event-3>", MessageD1),
								},
							},
						)
						Expect(err).ShouldNot(HaveOccurred())

						err = revisionStore.PrepareRevision(
							context.Background(),
							handlerID.Key,
							"<instance>",
							Revision{
								Begin: 2,
								End:   3,
								Events: []*envelopespec.Envelope{
									NewEnvelope("<event-4>", MessageE1),
								},
							},
						)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("applies only those events that occurred after destruction", func() {
						begin, end, snapshotAge, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(begin).To(BeNumerically("==", 2))
						Expect(end).To(BeNumerically("==", 4))
						Expect(snapshotAge).To(BeNumerically("==", 2))
						Expect(root.AppliedEvents).To(Equal(
							[]dogma.Message{
								MessageD1,
								MessageE1,
							},
						))
					})

					When("there is a snapshot available", func() {
						BeforeEach(func() {
							loader.SnapshotReader = snapshotReader
						})

						It("applies only those events that occurred after the snapshot", func() {
							err := snapshotStore.WriteSnapshot(
								context.Background(),
								handlerID.Key,
								"<instance>",
								&AggregateRoot{
									AppliedEvents: []dogma.Message{
										"<snapshot>",
									},
								},
								2,
							)
							Expect(err).ShouldNot(HaveOccurred())

							begin, end, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(begin).To(BeNumerically("==", 2))
							Expect(end).To(BeNumerically("==", 4))
							Expect(snapshotAge).To(BeNumerically("==", 1))
							Expect(root.AppliedEvents).To(Equal(
								[]dogma.Message{
									"<snapshot>",
									MessageE1,
								},
							))
						})

						It("ignores snapshots taken before destruction", func() {
							err := snapshotStore.WriteSnapshot(
								context.Background(),
								handlerID.Key,
								"<instance>",
								&AggregateRoot{
									AppliedEvents: []dogma.Message{
										"<snapshot>",
									},
								},
								0,
							)
							Expect(err).ShouldNot(HaveOccurred())

							begin, end, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(begin).To(BeNumerically("==", 2))
							Expect(end).To(BeNumerically("==", 4))
							Expect(snapshotAge).To(BeNumerically("==", 2))
							Expect(root.AppliedEvents).To(Equal(
								[]dogma.Message{
									MessageD1,
									MessageE1,
								},
							))
						})
					})
				})
			})
		})
	})
})

// revisionReaderStub is a test implementation of the RevisionReader interface.
type revisionReaderStub struct {
	RevisionReader

	ReadBoundsFunc func(
		ctx context.Context,
		hk, id string,
	) (begin, committed, end uint64, _ error)

	ReadRevisionsFunc func(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (revisions []Revision, _ error)
}

func (s *revisionReaderStub) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, committed, end uint64, _ error) {
	if s.ReadBoundsFunc != nil {
		return s.ReadBoundsFunc(ctx, hk, id)
	}

	if s.RevisionReader != nil {
		return s.RevisionReader.ReadBounds(ctx, hk, id)
	}

	return 0, 0, 0, nil
}

func (s *revisionReaderStub) ReadRevisions(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (revisions []Revision, _ error) {
	if s.ReadRevisionsFunc != nil {
		return s.ReadRevisionsFunc(ctx, hk, id, begin)
	}

	if s.RevisionReader != nil {
		return s.RevisionReader.ReadRevisions(ctx, hk, id, begin)
	}

	return nil, nil
}

// // eventWriterStub is a test implementation of the EventWriter interface.
// type eventWriterStub struct {
// 	EventWriter

// 	WriteEventsFunc func(
// 		ctx context.Context,
// 		hk, id string,
// 		begin, end uint64,
// 		events []*envelopespec.Envelope,
// 	) error
// }

// func (s *eventWriterStub) WriteEvents(
// 	ctx context.Context,
// 	hk, id string,
// 	begin, end uint64,
// 	events []*envelopespec.Envelope,
// ) error {
// 	if s.WriteEventsFunc != nil {
// 		return s.WriteEventsFunc(ctx, hk, id, begin, end, events)
// 	}

// 	if s.EventWriter != nil {
// 		return s.EventWriter.WriteEvents(ctx, hk, id, begin, end, events)
// 	}

// 	return nil
// }
