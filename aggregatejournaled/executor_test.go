package aggregate_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregatejournaled"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type CommandExecutor", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		journal *journalStub
		stream  *eventStreamStub
		loader  *Loader
		handler *AggregateMessageHandler

		command        parcel.Parcel
		event0, event1 parcel.Parcel
		ack            func(context.Context) error

		executor *CommandExecutor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		journal = &journalStub{
			Journal: &MemoryJournal{},
		}

		stream = &eventStreamStub{
			EventStream: &MemoryEventStream{},
		}

		loader = &Loader{
			Journal: journal,
			Stream:  stream,
		}

		handler = &AggregateMessageHandler{
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
				s.RecordEvent(MessageE2)
			},
		}

		command = NewParcel("<command>", MessageC1)
		event0 = parcel.Parcel{
			Envelope: &envelopespec.Envelope{
				MessageId:     "0",
				CausationId:   "<command>",
				CorrelationId: "<correlation>",
				SourceApplication: &envelopespec.Identity{
					Name: "<app-name>",
					Key:  "<app-key>",
				},
				SourceHandler: &envelopespec.Identity{
					Name: "<handler-name>",
					Key:  "<handler-key>",
				},
				SourceInstanceId: "<instance>",
				CreatedAt:        "2000-01-01T00:00:00Z",
				Description:      "{E1}",
				PortableName:     "MessageE",
				MediaType:        MessageE1Packet.MediaType,
				Data:             MessageE1Packet.Data,
			},
			Message:   MessageE1,
			CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		event1 = parcel.Parcel{
			Envelope: &envelopespec.Envelope{
				MessageId:     "1",
				CausationId:   "<command>",
				CorrelationId: "<correlation>",
				SourceApplication: &envelopespec.Identity{
					Name: "<app-name>",
					Key:  "<app-key>",
				},
				SourceHandler: &envelopespec.Identity{
					Name: "<handler-name>",
					Key:  "<handler-key>",
				},
				SourceInstanceId: "<instance>",
				CreatedAt:        "2000-01-01T00:00:01Z",
				Description:      "{E2}",
				PortableName:     "MessageE",
				MediaType:        MessageE2Packet.MediaType,
				Data:             MessageE2Packet.Data,
			},
			Message:   MessageE2,
			CreatedAt: time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC),
		}

		ack = func(ctx context.Context) error {
			return nil
		}

		executor = &CommandExecutor{
			HandlerIdentity: &envelopespec.Identity{
				Name: "<handler-name>",
				Key:  "<handler-key>",
			},
			InstanceID: "<instance>",
			Loader:     loader,
			Handler:    handler,
			Journal:    journal,
			Stream:     stream,
			Packer: NewPacker(
				message.TypeRoles{
					MessageCType: message.CommandRole,
					MessageEType: message.EventRole,
				},
			),
		}

		err := executor.Load(ctx)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("func ExecuteCommand()", func() {
		It("applies recorded events to the aggregate root", func() {
			err := executor.ExecuteCommand(ctx, command, ack)
			Expect(err).ShouldNot(HaveOccurred())

			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				_ dogma.AggregateCommandScope,
				_ dogma.Message,
			) {
				Expect(r).To(Equal(&AggregateRoot{
					AppliedEvents: []dogma.Message{
						MessageE1,
						MessageE2,
					},
				}))
			}
		})

		It("writes recorded events to the event stream", func() {
			err := executor.ExecuteCommand(ctx, command, ack)
			Expect(err).ShouldNot(HaveOccurred())

			expectEvents(
				ctx,
				stream,
				0,
				event0,
				event1,
			)
		})

		It("acknowledges the command", func() {
			acked := false
			ack = func(ctx context.Context) error {
				acked = true
				return nil
			}

			err := executor.ExecuteCommand(ctx, command, ack)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(acked).To(BeTrue())
		})

		When("there is a fault", func() {
			DescribeTable(
				"writes the events to the event stream exactly once",
				func(setup func()) {
					var (
						originalHandlerHandleCommand = handler.HandleCommandFunc
						originalJournalWrite         = journal.WriteFunc
						originalStreamWrite          = stream.WriteFunc
						originalAck                  = ack
					)

					By("creating a fault condition", func() {
						setup()
					})

					By("attempting execute the command", func() {
						defer func() {
							switch r := recover(); r {
							case nil:
								// no panic
							case "<panic>":
								// induced fault condition
							default:
								// any other panic
								panic(r)
							}
						}()

						err := executor.ExecuteCommand(ctx, command, ack)
						Expect(err).To(MatchError("<error>"))
					})

					By("resetting the fault condition", func() {
						handler.HandleCommandFunc = originalHandlerHandleCommand
						journal.WriteFunc = originalJournalWrite
						stream.WriteFunc = originalStreamWrite
						ack = originalAck
						executor.Packer = NewPacker(
							message.TypeRoles{
								MessageCType: message.CommandRole,
								MessageEType: message.EventRole,
							},
						)
					})

					By("reloading the snapshot", func() {
						err := executor.Load(ctx)
						Expect(err).ShouldNot(HaveOccurred())
					})

					By("retrying the command", func() {
						err := executor.ExecuteCommand(
							ctx,
							command,
							func(context.Context) error {
								return nil
							},
						)
						Expect(err).ShouldNot(HaveOccurred())
					})

					expectEvents(
						ctx,
						stream,
						0,
						event0,
						event1,
					)
				},
				Entry("entry cannot be written to the journal", func() {
					journal.WriteFunc = func(
						ctx context.Context,
						hk, id string,
						offset uint64,
						e JournalEntry,
					) error {
						return errors.New("<error>")
					}
				}),
				Entry("command cannot be acknowledged", func() {
					ack = func(context.Context) error {
						return errors.New("<error>")
					}
				}),
				Entry("events cannot be written to the stream", func() {
					stream.WriteFunc = func(
						ctx context.Context,
						ev parcel.Parcel,
					) error {
						return errors.New("<error>")
					}
				}),
				Entry("some events cannot be written to the stream", func() {
					stream.WriteFunc = func(
						ctx context.Context,
						ev parcel.Parcel,
					) error {
						// Setup stream to fail on the NEXT event.
						stream.WriteFunc = func(
							ctx context.Context,
							ev parcel.Parcel,
						) error {
							return errors.New("<error>")
						}

						// But succeed this time.
						return stream.EventStream.Write(ctx, ev)
					}
				}),
				Entry("handler panics", func() {
					handler.HandleCommandFunc = func(
						r dogma.AggregateRoot,
						s dogma.AggregateCommandScope,
						m dogma.Message,
					) {
						panic("<panic>")
					}
				}),
			)
		})
	})
})
