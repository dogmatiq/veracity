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

		command, event parcel.Parcel
		ack            func(context.Context) error

		handler  *AggregateMessageHandler
		journal  *journalStub
		stream   *eventStreamStub
		packer   *parcel.Packer
		executor *CommandExecutor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

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

		handler = &AggregateMessageHandler{
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
			},
		}

		journal = &journalStub{
			Journal: &MemoryJournal{},
		}

		stream = &eventStreamStub{
			EventStream: &MemoryEventStream{},
		}

		packer = NewPacker(
			message.TypeRoles{
				MessageCType: message.CommandRole,
				MessageEType: message.EventRole,
			},
		)

		executor = &CommandExecutor{
			HandlerIdentity: &envelopespec.Identity{
				Name: "<handler-name>",
				Key:  "<handler-key>",
			},
			InstanceID: "<instance>",
			Handler:    handler,
			Journal:    journal,
			Stream:     stream,
			Packer:     packer,
		}

		err := executor.Load(ctx)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("func ExecuteCommand()", func() {
		It("appends recorded events to the event stream", func() {
			err := executor.ExecuteCommand(ctx, command, ack)
			Expect(err).ShouldNot(HaveOccurred())

			expectEvents(
				ctx,
				stream,
				0,
				event,
			)
		})

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

		It("does not re-handle the command when acknowledgement fails", func() {
			err := executor.ExecuteCommand(
				ctx,
				command,
				func(context.Context) error {
					return errors.New("<error>")
				},
			)
			Expect(err).Should(MatchError("<error>"))

			acked := false
			executor = &CommandExecutor{
				HandlerIdentity: executor.HandlerIdentity,
				InstanceID:      executor.InstanceID,
				Handler:         executor.Handler,
				Journal:         executor.Journal,
				Stream:          executor.Stream,
				Packer:          executor.Packer,
			}

			err = executor.Load(ctx)
			Expect(err).ShouldNot(HaveOccurred())

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
