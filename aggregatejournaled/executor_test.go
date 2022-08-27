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
		packer  *parcel.Packer

		command, event parcel.Parcel
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
			},
		}

		packer = NewPacker(
			message.TypeRoles{
				MessageCType: message.CommandRole,
				MessageEType: message.EventRole,
			},
		)

		command = NewParcel("<command>", MessageC1)
		event = parcel.Parcel{
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
			Packer:     packer,
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
				event,
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

		When("an event cannot be written to the stream", func() {
			It("writes the event next time the instance is loaded", func() {
				stream.WriteFunc = func(
					ctx context.Context,
					ev parcel.Parcel,
				) error {
					return errors.New("<error>")
				}

				err := executor.ExecuteCommand(ctx, command, ack)
				Expect(err).Should(MatchError("<error>"))

				stream.WriteFunc = nil
				err = executor.Load(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				expectEvents(
					ctx,
					stream,
					0,
					event,
				)
			})
		})

		When("the command cannot be acknowledged", func() {
			It("does not re-handle the command", func() {
				err := executor.ExecuteCommand(
					ctx,
					command,
					func(context.Context) error {
						return errors.New("<error>")
					},
				)
				Expect(err).Should(MatchError("<error>"))

				err = executor.Load(ctx)
				Expect(err).ShouldNot(HaveOccurred())

				acked := false
				err = executor.ExecuteCommand(
					ctx,
					command,
					func(context.Context) error {
						acked = true
						return nil
					},
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(acked).To(BeTrue())

				expectEvents(
					ctx,
					stream,
					0,
					event,
				)
			})
		})
	})
})
