package aggregate_test

import (
	"context"
	"errors"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dodeca/logging"
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
)

var _ = Describe("type Worker", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		eventStore  *memory.AggregateEventStore
		eventReader *eventReaderStub
		eventWriter *eventWriterStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		packer   *parcel.Packer
		loader   *Loader
		commands chan Command
		handler  *AggregateMessageHandler
		logger   *logging.BufferedLogger
		worker   *Worker
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		DeferCleanup(cancel)

		eventStore = &memory.AggregateEventStore{}

		eventReader = &eventReaderStub{
			EventReader: eventStore,
		}

		eventWriter = &eventWriterStub{
			EventWriter: eventStore,
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

		loader = &Loader{
			EventReader:    eventReader,
			SnapshotReader: snapshotReader,
			Marshaler:      Marshaler,
		}

		commands = make(chan Command, DefaultCommandBuffer)

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

		logger = &logging.BufferedLogger{}

		worker = &Worker{
			WorkerConfig: WorkerConfig{
				Handler:         handler,
				HandlerIdentity: configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Packer:          packer,
				Loader:          loader,
				EventWriter:     eventWriter,
				SnapshotWriter:  snapshotWriter,
				Logger:          logger,
			},
			InstanceID: "<instance>",
			Commands:   commands,
		}
	})

	Describe("func Run()", func() {
		It("passes command messages to the handler", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				Expect(m).To(Equal(MessageC1))
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
			It("passes the handler an zero-valued aggregate root", func() {
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
				err := eventStore.WriteEvents(
					ctx,
					"<handler-key>",
					"<instance>",
					0,
					0,
					[]*envelopespec.Envelope{
						NewEnvelope("<existing>", MessageX1),
					},
				)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("passes the handler the correct aggregate root", func() {
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

		It("persists recorded events", func() {
			go func() {
				defer GinkgoRecover()

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-1>", MessageC1),
				)

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC2),
				)

				cancel()
			}()

			err := worker.Run(ctx)
			Expect(err).To(Equal(context.Canceled))

			expectEvents(
				eventStore,
				"<handler-key>",
				"<instance>",
				0,
				[]*envelopespec.Envelope{
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
			)
		})

		It("returns an error when there is an OCC failure", func() {
			go func() {
				defer GinkgoRecover()

				By("executing a command to ensure the worker is running")

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-1>", MessageC1),
				)

				By("writing an event to the event store that the worker doesn't know about")

				err := eventStore.WriteEvents(
					ctx,
					"<handler-key>",
					"<instance>",
					0,
					1,
					[]*envelopespec.Envelope{
						NewEnvelope("<existing>", MessageX1),
					},
				)
				Expect(err).ShouldNot(HaveOccurred())

				By("sending a second command")

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC1),
				)
			}()

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"cannot write events for aggregate root <handler-name>[<instance>]: optimistic concurrency conflict, 1 is not the next revision",
				),
			)
		})

		It("writes a snapshot when the snapshot interval is exceeded", func() {
			worker.SnapshotInterval = 2

			go func() {
				defer GinkgoRecover()

				By("sending a command")

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-1>", MessageC1),
				)

				By("ensuring no snapshot has been taken")

				_, ok, err := snapshotStore.ReadSnapshot(
					ctx,
					"<handler-key>",
					"<instance>",
					&AggregateRoot{},
					0,
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeFalse())

				By("sending a second command")

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC2),
				)

				By("ensuring that a snapshot has been written")

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

				By("sending a third command")

				executeCommandSync(
					ctx,
					commands,
					NewParcel("<command-2>", MessageC2),
				)

				By("ensuring that no newer snapshot has been taken")

				snapshotRev, ok, err = snapshotStore.ReadSnapshot(
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
					s.Destroy()
				}

				err := eventStore.WriteEvents(
					ctx,
					"<handler-key>",
					"<instance>",
					0,
					0,
					[]*envelopespec.Envelope{
						NewEnvelope("<existing-1>", MessageX1),
						NewEnvelope("<existing-2>", MessageX2),
					},
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

			It("archives historical events and snapshots, and returns nil", func() {
				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				begin, end, err := eventStore.ReadBounds(
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

			It("archives newly recorded events", func() {
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

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				begin, end, err := eventStore.ReadBounds(
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
						executeCommandSync(
							ctx,
							commands,
							NewParcel("<command>", MessageC1),
						)

						cancel()
					}()

					err := worker.Run(ctx)
					Expect(err).To(Equal(context.Canceled))

					begin, end, err := eventStore.ReadBounds(
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
				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

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

			It("returns nil", func() {
				err := worker.Run(ctx)
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

			err := worker.Run(ctx)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns an error if the root cannot be loaded", func() {
			eventReader.ReadBoundsFunc = func(
				ctx context.Context,
				hk, id string,
			) (uint64, uint64, error) {
				return 0, 0, errors.New("<error>")
			}

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read event revision bounds: <error>",
				),
			)
		})

		It("returns an error if events cannot be written", func() {
			eventWriter.WriteEventsFunc = func(
				ctx context.Context,
				hk, id string,
				begin, end uint64,
				events []*envelopespec.Envelope,
			) error {
				return errors.New("<error>")
			}

			executeCommandAsync(
				ctx,
				commands,
				NewParcel("<command>", MessageC1),
			)

			err := worker.Run(ctx)
			Expect(err).To(
				MatchError(
					"cannot write events for aggregate root <handler-name>[<instance>]: <error>",
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

			Expect(logger.Messages()).To(ContainElement(
				logging.BufferedLogMessage{
					Message: "aggregate <handler-name>[<instance>]: <log-message 1 2 3>",
				},
			))
		})
	})
})
