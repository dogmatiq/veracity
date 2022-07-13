package aggregate_test

import (
	"context"
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
	. "github.com/jmalloc/gomegax"
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

		commands = make(chan Command, 5)

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
				EventWriter:     eventWriter,
				SnapshotWriter:  snapshotWriter,
			},
			InstanceID: "<instance>",
			Commands:   commands,
		}
	})

	Describe("func Run()", func() {
		When("a command is received", func() {
			It("passes the command message to the handler", func() {
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
						false, // archive
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

			It("returns an error when there is an OCC failure due to a nextOffset mismatch", func() {
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
						false, // archive
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
						`optimistic concurrency conflict, 1 is not the next offset`,
					),
				)
			})

			It("returns an error where there is an OCC failure due to a firstOffset mismatch", func() {
				go func() {
					defer GinkgoRecover()

					By("executing a command to ensure the worker is running")

					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command-1>", MessageC1),
					)

					By("archiving existing events without the worker's knowledge")

					err := eventStore.WriteEvents(
						ctx,
						"<handler-key>",
						"<instance>",
						0,
						1,
						nil,  // no events
						true, // archive
					)
					Expect(err).ShouldNot(HaveOccurred())

					By("sending a second command")

					select {
					case <-ctx.Done():
						Expect(ctx.Err()).ShouldNot(HaveOccurred())
					case commands <- Command{
						Context: ctx,
						Parcel:  NewParcel("<command-2>", MessageC1),
						Result:  make(chan error),
					}:
					}
				}()

				err := worker.Run(ctx)
				Expect(err).To(
					MatchError(
						`optimistic concurrency conflict, 0 is not the first offset`,
					),
				)
			})
		})

		When("the instance is destroyed", func() {
			BeforeEach(func() {
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
					false, // archive
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
				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					s.Destroy()
				}

				executeCommandAsync(
					ctx,
					commands,
					NewParcel("<command>", MessageC1),
				)

				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				firstOffset, nextOffset, err := eventStore.ReadBounds(
					ctx,
					"<handler-key>",
					"<instance>",
				)
				Expect(firstOffset).To(BeNumerically("==", 2))
				Expect(nextOffset).To(BeNumerically("==", 2))

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

				firstOffset, nextOffset, err := eventStore.ReadBounds(
					ctx,
					"<handler-key>",
					"<instance>",
				)
				Expect(firstOffset).To(BeNumerically("==", 3))
				Expect(nextOffset).To(BeNumerically("==", 3))
			})

			XIt("returns nil", func() {
			})
		})

		When("the idle timeout is reached", func() {
			BeforeEach(func() {
				worker.IdleTimeout = 1 * time.Millisecond
			})

			XIt("takes a snapshot if the existing snapshot is out-of-date", func() {
			})

			XIt("does not take a snapshot if the existing snapshot is up-to-date", func() {
			})

			It("returns nil", func() {
				err := worker.Run(ctx)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		XIt("honours the snapshot write timeout", func() {

		})

		XIt("does not return an error if writing a snapshot fails", func() {

		})

		XIt("returns an error if the root cannot be loaded", func() {
		})

		XIt("returns an error if events cannot be written", func() {
		})

		XIt("returns an error if the context is canceled while waiting for a command", func() {
		})

		XIt("returns an error if the context is canceled while persisting a snapshot", func() {
		})
	})
})

// expectEvents reads all events from store starting from offset and asserts
// that they are equal to expectedEvents.
func expectEvents(
	reader EventReader,
	hk, id string,
	offset uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, more, err := reader.ReadEvents(
			context.Background(),
			hk,
			id,
			offset,
		)
		Expect(err).ShouldNot(HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if !more {
			break
		}

		offset += uint64(len(events))
	}

	Expect(producedEvents).To(EqualX(expectedEvents))
}

// executeCommandSync executes a command by sending it to a command channel and
// waiting for it to complete.
func executeCommandSync(
	ctx context.Context,
	commands chan<- Command,
	command parcel.Parcel,
) {
	result := executeCommandAsync(ctx, commands, command)

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case err := <-result:
		Expect(err).ShouldNot(HaveOccurred())
	}
}

// executeCommandSync executes a command by sending it to a command channel.
func executeCommandAsync(
	ctx context.Context,
	commands chan<- Command,
	command parcel.Parcel,
) <-chan error {
	result := make(chan error, 1)

	cmd := Command{
		Context: ctx,
		Parcel:  command,
		Result:  result,
	}

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	return result
}
