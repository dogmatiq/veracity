package aggregate_test

import (
	"context"
	"errors"
	"time"

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

var _ = Describe("type Loader", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		revisionStore  *memory.AggregateRevisionStore
		revisionReader *revisionReaderStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		handlerID configkit.Identity
		root      *AggregateRoot
		loader    *Loader
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

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

		loader = &Loader{
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
			) (Bounds, error) {
				return Bounds{}, errors.New("<error>")
			}

			_, _, err := loader.Load(
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
				bounds, snapshotAge, err := loader.Load(
					context.Background(),
					handlerID,
					"<instance>",
					root,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(bounds).To(Equal(
					Bounds{
						Begin:     0,
						End:       0,
						Committed: 0,
					},
				))
				Expect(snapshotAge).To(BeNumerically("==", 0))
				Expect(root.AppliedEvents).To(BeEmpty())
			})
		})

		When("the instance has revisions", func() {
			BeforeEach(func() {
				commitRevision(
					ctx,
					revisionStore,
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
			})

			When("there is no snapshot reader", func() {
				It("applies all historical events to the root", func() {
					bounds, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(bounds).To(Equal(
						Bounds{
							Begin:     0,
							End:       1,
							Committed: 1,
						},
					))
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

					_, _, err := loader.Load(
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

					bounds, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(bounds).To(Equal(
						Bounds{
							Begin:     0,
							End:       1,
							Committed: 1,
						},
					))
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
							bounds, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(bounds).To(Equal(
								Bounds{
									Begin:     0,
									End:       1,
									Committed: 1,
								},
							))
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
							commitRevision(
								ctx,
								revisionStore,
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
						})

						It("applies only those events that occurred after the snapshot", func() {
							bounds, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(bounds).To(Equal(
								Bounds{
									Begin:     0,
									End:       2,
									Committed: 2,
								},
							))
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

						_, _, err := loader.Load(
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
						_, _, err := loader.Load(
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

						_, _, err := loader.Load(
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

						_, _, err := loader.Load(
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

						_, _, err := loader.Load(
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

						_, _, err := loader.Load(
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
					commitRevision(
						ctx,
						revisionStore,
						handlerID.Key,
						"<instance>",
						Revision{
							Begin: 2,
							End:   1,
						},
					)
				})

				It("does not apply any events", func() {
					bounds, snapshotAge, err := loader.Load(
						context.Background(),
						handlerID,
						"<instance>",
						root,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(bounds).To(Equal(
						Bounds{
							Begin:     2,
							End:       2,
							Committed: 2,
						},
					))
					Expect(snapshotAge).To(BeNumerically("==", 0))
					Expect(root.AppliedEvents).To(BeEmpty())
				})

				When("the instance has been recreated", func() {
					BeforeEach(func() {
						commitRevision(
							ctx,
							revisionStore,
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

						commitRevision(
							ctx,
							revisionStore,
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
					})

					It("applies only those events that occurred after destruction", func() {
						bounds, snapshotAge, err := loader.Load(
							context.Background(),
							handlerID,
							"<instance>",
							root,
						)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(bounds).To(Equal(
							Bounds{
								Begin:     2,
								End:       4,
								Committed: 4,
							},
						))
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

							bounds, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(bounds).To(Equal(
								Bounds{
									Begin:     2,
									End:       4,
									Committed: 4,
								},
							))
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

							bounds, snapshotAge, err := loader.Load(
								context.Background(),
								handlerID,
								"<instance>",
								root,
							)
							Expect(err).ShouldNot(HaveOccurred())
							Expect(bounds).To(Equal(
								Bounds{
									Begin:     2,
									End:       4,
									Committed: 4,
								},
							))
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
