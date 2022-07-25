package aggregate_test

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("type Worker", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		revisionStore  *memory.AggregateRevisionStore
		revisionReader *revisionReaderStub
		revisionWriter *revisionWriterStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		packer   *parcel.Packer
		loader   *Loader
		commands chan *Command
		idle     chan string
		handler  *AggregateMessageHandler
		worker   *Worker
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		revisionStore = &memory.AggregateRevisionStore{}

		revisionReader = &revisionReaderStub{
			RevisionReader: revisionStore,
		}

		revisionWriter = &revisionWriterStub{
			RevisionWriter: revisionStore,
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

		logger, err := zap.NewDevelopment(
			zap.AddStacktrace(zap.PanicLevel + 1),
		)
		Expect(err).ShouldNot(HaveOccurred())

		loader = &Loader{
			RevisionReader: revisionReader,
			SnapshotReader: snapshotReader,
			Marshaler:      Marshaler,
			Logger:         logger,
		}

		commands = make(chan *Command, DefaultCommandBuffer)
		idle = make(chan string)

		handler = &AggregateMessageHandler{
			ConfigureFunc: func(c dogma.AggregateConfigurer) {
				c.Identity("<handler-name>", "<handler-key>")
				c.ConsumesCommandType(MessageC{})
				c.ProducesEventType(MessageE{})
			},
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				switch m {
				case MessageC1:
					s.RecordEvent(MessageE1)
				case MessageC2:
					s.RecordEvent(MessageE2)
				}
			},
		}

		packer = NewPacker(
			message.TypeRoles{
				MessageCType: message.CommandRole,
				MessageEType: message.EventRole,
			},
		)

		worker = &Worker{
			WorkerConfig: WorkerConfig{
				Handler:         handler,
				HandlerIdentity: configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Packer:          packer,
				Loader:          loader,
				RevisionWriter:  revisionWriter,
				SnapshotWriter:  snapshotWriter,
				Logger:          logger,
			},
			InstanceID: "<instance>",
			Commands:   commands,
			Idle:       idle,
		}
	})

	// shutdownWorkerWhenIdle simulates the supervisor shutting down the worker
	// when it enters the idle state.
	shutdownWorkerWhenIdle := func() {
		go func() {
			select {
			case <-ctx.Done():
			case <-idle:
				close(commands)
			}
		}()
	}

	Describe("func Run()", func() {
		It("ACKs the command if it is handled successfully", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				Expect(m).To(Equal(MessageC1))
				cancel()
			}

			cmd := executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))

			Eventually(cmd.Done()).Should(BeClosed())
			Expect(cmd.Err()).ShouldNot(HaveOccurred())
		})

		It("NACKs the command if preparing the revision fails", func() {
			revisionWriter.PrepareRevisionFunc = func(
				ctx context.Context,
				hk, id string,
				rev Revision,
			) error {
				return errors.New("<error>")
			}

			cmd := executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"cannot prepare revision 0 of aggregate root <handler-name>[<instance>]: <error>",
				),
			)

			Eventually(cmd.Done()).Should(BeClosed())
			Expect(cmd.Err()).To(MatchError("shutting down"))
		})

		It("applies recorded events to the aggregate root", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)

				x := r.(*AggregateRoot)
				Expect(x.AppliedEvents).To(ConsistOf(
					MessageE1,
				))

				cancel()
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
		})

		When("the instance has no historical events", func() {
			It("passes the handler a new aggregate root", func() {
				expect := handler.New()

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					Expect(r).To(Equal(expect))
					cancel()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				err := worker.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})
		})

		When("the instance has historical events", func() {
			BeforeEach(func() {
				rev := Revision{
					Begin: 0,
					End:   0,
					Events: []*envelopespec.Envelope{
						NewEnvelope("<existing>", MessageX1),
					},
				}

				err := revisionStore.PrepareRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = revisionStore.CommitRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("passes the handler the the aggregate root loaded via the loader", func() {
				expect := handler.New()
				expect.ApplyEvent(MessageX1)

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					Expect(r).To(Equal(expect))
					cancel()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				err := worker.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})
		})

		It("persists a new revision", func() {
			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command-1>", MessageC1),
			)

			cmd := executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command-2>", MessageC2),
			)

			go func() {
				// Shutdown the worker when the command has been ACKd.
				select {
				case <-ctx.Done():
				case <-cmd.Done():
					cancel()
				}
			}()

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))

			expectRevisions(
				ctx,
				revisionStore,
				"<handler-key>",
				"<instance>",
				0,
				[]Revision{
					{
						Begin: 0,
						End:   0,
						Events: []*envelopespec.Envelope{
							{
								MessageId:         "0",
								CausationId:       "<command-1>",
								CorrelationId:     "<correlation>",
								SourceApplication: packer.Application,
								SourceHandler:     marshalkit.MustMarshalEnvelopeIdentity(worker.HandlerIdentity),
								SourceInstanceId:  "<instance>",
								CreatedAt:         "2000-01-01T00:00:00Z",
								Description:       "{E1}",
								PortableName:      MessageEPortableName,
								MediaType:         MessageE1Packet.MediaType,
								Data:              MessageE1Packet.Data,
							},
						},
					},
					{
						Begin: 0,
						End:   1,
						Events: []*envelopespec.Envelope{
							{
								MessageId:         "1",
								CausationId:       "<command-2>",
								CorrelationId:     "<correlation>",
								SourceApplication: packer.Application,
								SourceHandler:     marshalkit.MustMarshalEnvelopeIdentity(worker.HandlerIdentity),
								SourceInstanceId:  "<instance>",
								CreatedAt:         "2000-01-01T00:00:01Z",
								Description:       "{E2}",
								PortableName:      MessageEPortableName,
								MediaType:         MessageE2Packet.MediaType,
								Data:              MessageE2Packet.Data,
							},
						},
					},
				},
			)
		})

		It("returns an error when there is an OCC failure", func() {
			// Send a command that we can wait on to ensure that the worker has
			// loaded the aggregate state.
			cmd := executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command-1>", MessageC1),
			)

			go func() {
				defer GinkgoRecover()

				Eventually(cmd.Done()).Should(BeClosed())

				// Write an extra revision to the store that the worker doesn't
				// know about, making the worker's in-memory knowledge of the
				// current revision out-of-date.
				err := revisionStore.PrepareRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					Revision{
						Begin: 0,
						End:   1,
						Events: []*envelopespec.Envelope{
							NewEnvelope("<existing>", MessageX1),
						},
					},
				)
				Expect(err).ShouldNot(HaveOccurred())

				// Send a second command to trigger the OCC failure.
				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC1),
				)
			}()

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"cannot prepare revision 1 of aggregate root <handler-name>[<instance>]: optimistic concurrency conflict, 1 is not the next revision",
				),
			)
		})

		It("writes a snapshot when the snapshot interval is exceeded", func() {
			// Take a snapshot after every other revision.
			worker.SnapshotInterval = 2

			// Make revision 0.
			cmd := executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command-1>", MessageC1),
			)

			go func() {
				defer GinkgoRecover()

				// Wait for revision 0 to be created.
				Eventually(cmd.Done()).Should(BeClosed())

				// Ensure no snapshot has been taken.
				_, ok, err := snapshotStore.ReadSnapshot(
					ctx,
					"<handler-key>",
					"<instance>",
					&AggregateRoot{},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeFalse())

				// Make revision 1.
				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC1),
				)

				// Make revision 2.
				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC1),
				)

				// Ensure that the snapshot was taken at revision 1.
				snapshotRev, ok, err := snapshotStore.ReadSnapshot(
					ctx,
					"<handler-key>",
					"<instance>",
					&AggregateRoot{},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(snapshotRev).To(BeNumerically("==", 1))

				cancel()
			}()

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
		})

		When("the instance is destroyed", func() {
			BeforeEach(func() {
				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					s.RecordEvent(MessageE1)
					s.Destroy()
				}

				rev := Revision{
					Begin: 0,
					End:   0,
					Events: []*envelopespec.Envelope{
						NewEnvelope("<existing-1>", MessageX1),
						NewEnvelope("<existing-2>", MessageX2),
					},
				}

				err := revisionStore.PrepareRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = revisionStore.CommitRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = snapshotStore.WriteSnapshot(
					context.Background(),
					"<handler-key>",
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

			It("archives historical revisions and snapshots, and signals the idle state", func() {
				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				begin, _, end, err := revisionStore.ReadBounds(
					ctx,
					"<handler-key>",
					"<instance>",
				)
				Expect(begin).To(BeNumerically("==", 2))
				Expect(end).To(BeNumerically("==", 2))

				_, ok, err := snapshotStore.ReadSnapshot(
					ctx,
					"<handler-key>",
					"<instance>",
					&AggregateRoot{},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			It("does not return if there are commands waiting to be handled", func() {
				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					if m == MessageC1 {
						s.Destroy()
						return
					}

					Expect(m).To(Equal(MessageC2))
					cancel()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC2),
				)

				err := worker.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})

			It("archives the new revision", func() {
				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					s.RecordEvent(MessageE3)
					s.Destroy()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				begin, _, end, err := revisionStore.ReadBounds(
					ctx,
					"<handler-key>",
					"<instance>",
				)
				Expect(begin).To(BeNumerically("==", 2))
				Expect(end).To(BeNumerically("==", 2))
			})

			It("returns an error if the context is canceled while archiving a snapshot", func() {
				snapshotWriter.ArchiveSnapshotsFunc = func(
					ctx context.Context,
					hk, id string,
				) error {
					cancel()
					return ctx.Err()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				err := worker.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})

			It("does not return an error if archiving snapshots fails", func() {
				snapshotWriter.ArchiveSnapshotsFunc = func(
					ctx context.Context,
					hk, id string,
				) error {
					return errors.New("<error>")
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("does not return an error if the SnapshotWriter is nil", func() {
				worker.SnapshotWriter = nil

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())
			})

			When("events are recorded after the Destroy() is called", func() {
				It("archives neither events nor snapshots", func() {
					handler.HandleCommandFunc = func(
						r dogma.AggregateRoot,
						s dogma.AggregateCommandScope,
						m dogma.Message,
					) {
						s.Destroy()
						s.RecordEvent(MessageE3)
					}

					go func() {
						defer GinkgoRecover()

						executeCommandSync(
							ctx,
							commands,
							NewParcel("<command>", MessageC1),
						)

						cancel()
					}()

					err := worker.Run(ctx)
					Expect(err).To(Equal(context.Canceled))

					begin, _, end, err := revisionStore.ReadBounds(
						ctx,
						"<handler-key>",
						"<instance>",
					)
					Expect(begin).To(BeNumerically("==", 0))
					Expect(end).To(BeNumerically("==", 2))

					snapshotRev, ok, err := snapshotStore.ReadSnapshot(
						ctx,
						"<handler-key>",
						"<instance>",
						&AggregateRoot{},
						0,
					)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(snapshotRev).To(BeNumerically("==", 0))
				})
			})
		})

		When("the idle timeout is exceeded", func() {
			BeforeEach(func() {
				worker.IdleTimeout = 5 * time.Millisecond
			})

			It("takes a snapshot if the existing snapshot is out-of-date", func() {
				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				snapshotRev, ok, err := snapshotStore.ReadSnapshot(
					ctx,
					"<handler-key>",
					"<instance>",
					&AggregateRoot{},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(snapshotRev).To(BeNumerically("==", 0))
			})

			It("does not take a snapshot if the existing snapshot is up-to-date", func() {
				rev := Revision{
					Begin: 0,
					End:   0,
					Events: []*envelopespec.Envelope{
						NewEnvelope("<existing-1>", MessageX1),
						NewEnvelope("<existing-2>", MessageX2),
					},
				}

				err := revisionStore.PrepareRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = revisionStore.CommitRevision(
					ctx,
					"<handler-key>",
					"<instance>",
					rev,
				)
				Expect(err).ShouldNot(HaveOccurred())

				err = snapshotStore.WriteSnapshot(
					context.Background(),
					"<handler-key>",
					"<instance>",
					&AggregateRoot{
						AppliedEvents: []dogma.Message{
							"<snapshot>",
						},
					},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					// Don't record an event
				}

				snapshotWriter.WriteSnapshotFunc = func(
					ctx context.Context,
					hk, id string,
					r dogma.AggregateRoot,
					rev uint64,
				) error {
					panic("unexpected call")
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				go shutdownWorkerWhenIdle()

				err = worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		It("does not return an error if writing a snapshot fails", func() {
			// Rely on the fact that a snapshot is taken when the worker shuts
			// down due to idle timeout.
			worker.IdleTimeout = 5 * time.Millisecond

			called := false
			snapshotWriter.WriteSnapshotFunc = func(
				ctx context.Context,
				hk, id string,
				r dogma.AggregateRoot,
				rev uint64,
			) error {
				called = true
				return errors.New("<error>")
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			go shutdownWorkerWhenIdle()

			err := worker.Run(ctx)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		It("does not return an error if the SnapshotWriter is nil", func() {
			// Rely on the fact that a snapshot is taken when the worker shuts
			// down due to idle timeout.
			worker.IdleTimeout = 5 * time.Millisecond
			worker.SnapshotWriter = nil

			snapshotWriter.WriteSnapshotFunc = func(
				ctx context.Context,
				hk, id string,
				r dogma.AggregateRoot,
				rev uint64,
			) error {
				panic("unexpected call")
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			go shutdownWorkerWhenIdle()

			err := worker.Run(ctx)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns an error if the root cannot be loaded", func() {
			revisionReader.ReadBoundsFunc = func(
				ctx context.Context,
				hk, id string,
			) (uint64, uint64, uint64, error) {
				return 0, 0, 0, errors.New("<error>")
			}

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read revision bounds: <error>",
				),
			)
		})

		It("returns an error if the context is canceled while waiting for a command", func() {
			// This test relies on the fact that the memory-based persistence
			// being used does not use the context.
			cancel()
			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
		})

		It("returns an error if the context is canceled while persisting a snapshot", func() {
			// Rely on the fact that a snapshot is taken when the worker shuts
			// down due to idle timeout.
			worker.IdleTimeout = 5 * time.Millisecond

			snapshotWriter.WriteSnapshotFunc = func(
				ctx context.Context,
				hk, id string,
				r dogma.AggregateRoot,
				rev uint64,
			) error {
				cancel()
				return ctx.Err()
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
		})

		It("makes the instance ID available via the scope", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				Expect(s.InstanceID()).To(Equal("<instance>"))
				cancel()
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
		})

		It("allows logging via the scope", func() {
			buffer := &bytes.Buffer{}

			logger := zap.New(
				zapcore.NewCore(
					zapcore.NewConsoleEncoder(
						zap.NewDevelopmentEncoderConfig(),
					),
					zapcore.AddSync(buffer),
					zapcore.DebugLevel,
				),
			)

			worker.Logger = logger

			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.Log("<log-message %d %d %d>", 1, 2, 3)
				cancel()
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))

			Expect(buffer.String()).To(ContainSubstring(
				"<log-message 1 2 3>",
			))
		})
	})
})
