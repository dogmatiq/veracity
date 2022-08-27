package aggregate_test

import (
	"context"
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

		// journal *journalStub
		stream   *eventStreamStub
		ack      func(ctx context.Context) error
		executor *CommandExecutor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		stream = &eventStreamStub{
			EventStream: &MemoryEventStream{},
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
			Stream:     stream,
			Packer: NewPacker(
				message.TypeRoles{
					MessageCType: message.CommandRole,
					MessageEType: message.EventRole,
				},
			),
			Root: &AggregateRoot{},
			HandleCommand: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
			},
		}
	})

	Describe("func ExecuteCommand()", func() {
		It("appends recorded events to the event stream", func() {
			cmd := NewParcel("<command>", MessageC1)
			err := executor.ExecuteCommand(ctx, cmd, ack)
			Expect(err).ShouldNot(HaveOccurred())

			expectEvents(
				ctx,
				stream,
				0,
				[]parcel.Parcel{
					{
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
					},
				},
			)
		})

		It("applies recorded events to the aggregate root", func() {
			cmd := NewParcel("<command>", MessageC1)
			err := executor.ExecuteCommand(ctx, cmd, ack)
			Expect(err).ShouldNot(HaveOccurred())

			executor.HandleCommand = func(
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
			called := false
			ack = func(ctx context.Context) error {
				called = true
				return nil
			}

			cmd := NewParcel("<command>", MessageC1)
			err := executor.ExecuteCommand(ctx, cmd, ack)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(called).To(BeTrue())
		})
	})
})
