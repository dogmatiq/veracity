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

var _ = Describe("type CommandExecutor (fault recovery)", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		journal *journalStub
		stream  *eventStreamStub
		handler *AggregateMessageHandler
		ackErr  error

		command        parcel.Parcel
		event0, event1 parcel.Parcel
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
	})

	Describe("func ExecuteCommand()", func() {
		DescribeTable(
			"writes the events to the event stream exactly once",
			func(setup func()) {
				acked := false
				ack := func(ctx context.Context) error {
					if ackErr != nil {
						return ackErr
					}
					acked = true
					return nil
				}

				newExecutor := func() *CommandExecutor {
					return &CommandExecutor{
						HandlerIdentity: &envelopespec.Identity{
							Name: "<handler-name>",
							Key:  "<handler-key>",
						},
						InstanceID: "<instance>",
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
				}

				var executor *CommandExecutor

				By("initializing the executor", func() {
					executor = newExecutor()
					err := executor.Load(ctx)
					Expect(err).ShouldNot(HaveOccurred())
				})

				originalJournalWriteFunc := journal.WriteFunc
				originalStreamWriteFunc := stream.WriteFunc
				originalStreamWriteAtOffsetFunc := stream.WriteAtOffsetFunc
				originalHandlerHandleCommandFunc := handler.HandleCommandFunc
				originalAckErr := ackErr

				By("creating a fault condition", func() {
					setup()
				})

				By("attempting to execute the command", func() {
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
					journal.WriteFunc = originalJournalWriteFunc
					stream.WriteFunc = originalStreamWriteFunc
					stream.WriteAtOffsetFunc = originalStreamWriteAtOffsetFunc
					handler.HandleCommandFunc = originalHandlerHandleCommandFunc
					ackErr = originalAckErr
				})

				By("initializing new executor", func() {
					executor = newExecutor()
					err := executor.Load(ctx)
					Expect(err).ShouldNot(HaveOccurred())
				})

				if !acked {
					By("retrying unacknowledged the command", func() {
						err := executor.ExecuteCommand(ctx, command, ack)
						Expect(err).ShouldNot(HaveOccurred())
					})
				}

				Expect(acked).To(BeTrue(), "command not acknowledged")

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
				ackErr = errors.New("<error>")
			}),
			Entry("events cannot be written to the stream", func() {
				stream.WriteFunc = func(
					ctx context.Context,
					ev parcel.Parcel,
				) error {
					return errors.New("<error>")
				}

				stream.WriteAtOffsetFunc = func(
					ctx context.Context,
					offset uint64,
					ev parcel.Parcel,
				) error {
					return errors.New("<error>")
				}
			}),
			Entry("some events cannot be written to the stream", func() {
				succeed := true

				stream.WriteFunc = func(
					ctx context.Context,
					ev parcel.Parcel,
				) error {
					if succeed {
						succeed = false
						return stream.EventStream.Write(ctx, ev)
					}

					return errors.New("<error>")
				}

				stream.WriteAtOffsetFunc = func(
					ctx context.Context,
					offset uint64,
					ev parcel.Parcel,
				) error {
					if succeed {
						succeed = false
						return stream.EventStream.WriteAtOffset(ctx, offset, ev)
					}

					return errors.New("<error>")
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
