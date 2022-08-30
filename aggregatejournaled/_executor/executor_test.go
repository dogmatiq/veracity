package executor_test

import (
	"context"
	"time"

	// . "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"

	// . "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregatejournaled/executor"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Executor", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		handler *AggregateMessageHandler
		journ   *journal.Stub[JournalEntry]
		// stream  *eventStreamStub

		command parcel.Parcel
		// event   parcel.Parcel

		acked []string

		executor *Executor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		handler = &AggregateMessageHandler{}

		journ = &journal.Stub[JournalEntry]{
			Journal: &journal.InMemory[JournalEntry]{},
		}

		// stream = &eventStreamStub{
		// 	EventStream: &MemoryEventStream{},
		// }

		command = NewParcel("<command>", MessageC1)
		// event = parcel.Parcel{
		// 	Envelope: &envelopespec.Envelope{
		// 		MessageId:     "0",
		// 		CausationId:   "<command>",
		// 		CorrelationId: "<correlation>",
		// 		SourceApplication: &envelopespec.Identity{
		// 			Name: "<app-name>",
		// 			Key:  "<app-key>",
		// 		},
		// 		SourceHandler: &envelopespec.Identity{
		// 			Name: "<handler-name>",
		// 			Key:  "<handler-key>",
		// 		},
		// 		SourceInstanceId: "<instance>",
		// 		CreatedAt:        "2000-01-01T00:00:00Z",
		// 		Description:      "{E1}",
		// 		PortableName:     "MessageE",
		// 		MediaType:        MessageE1Packet.MediaType,
		// 		Data:             MessageE1Packet.Data,
		// 	},
		// 	Message:   MessageE1,
		// 	CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		// }

		acked = nil

		executor = &Executor{
			Handler: handler,
			Journal: journ,
			Ack: func(ctx context.Context, id string) error {
				acked = append(acked, id)
				return nil
			},
		}
	})

	Describe("func ExecuteCommand()", func() {
		It("invokes the handler", func() {
			called := false
			handler.HandleCommandFunc = func(
				_ dogma.AggregateRoot,
				_ dogma.AggregateCommandScope,
				_ dogma.Message,
			) {
				called = true
			}

			err := executor.ExecuteCommand(ctx, command)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(called).To(BeTrue())
		})

		It("applies recorded events to the aggregate root", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				_ dogma.Message,
			) {
				s.RecordEvent(MessageE1)

				Expect(r).To(Equal(&AggregateRoot{
					AppliedEvents: []dogma.Message{
						MessageE1,
					},
				}))
			}

			err := executor.ExecuteCommand(ctx, command)
			Expect(err).ShouldNot(HaveOccurred())
		})

		// It("writes recorded events to the event stream", func() {
		// 	err := executor.ExecuteCommand(ctx, command)
		// 	Expect(err).ShouldNot(HaveOccurred())

		// 	expectEvents(
		// 		ctx,
		// 		stream,
		// 		0,
		// 		event,
		// 	)
		// })

		It("acknowledges the command", func() {
			err := executor.ExecuteCommand(ctx, command)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(acked).To(ConsistOf("<command>"))
		})
	})
})
