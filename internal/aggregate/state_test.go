package aggregate_test

import (
	"context"
	"sync"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/internal/aggregate"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/eventstream/eventstreamtest"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type CommandExecutor (aggregate root state)", func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		packer        *envelope.Packer
		handler       *AggregateMessageHandler
		journalOpener *memory.JournalOpener[*JournalRecord]
		events        *eventstream.EventStream
		executor      *CommandExecutor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()
		handler = &AggregateMessageHandler{}
		journalOpener = &memory.JournalOpener[*JournalRecord]{}

		events = &eventstream.EventStream{
			Journal: &memory.Journal[*eventstream.JournalRecord]{},
			Logger:  zapx.NewTesting("eventstream"),
		}

		executor = &CommandExecutor{
			HandlerIdentity: &envelopespec.Identity{
				Name: "<handler-name>",
				Key:  "<handler-key>",
			},
			Handler:       handler,
			Packer:        packer,
			JournalOpener: journalOpener,
			EventAppender: events,
			Logger:        zapx.NewTesting("<handler-name>"),
		}
	})

	It("applies newly recorded events to the aggregate root immediately", func() {
		handler.HandleCommandFunc = func(
			r dogma.AggregateRoot,
			s dogma.AggregateCommandScope,
			m dogma.Message,
		) {
			defer GinkgoRecover()
			stub := r.(*AggregateRoot)

			s.RecordEvent(MessageE1)
			Expect(stub.AppliedEvents).To(ConsistOf(MessageE1))
		}

		run(
			ctx,
			executor,
			func(ctx context.Context) error {
				return executor.ExecuteCommand(
					ctx,
					"<instance-id>",
					packer.Pack(MessageC1),
				)
			},
		)
	})

	It("applies historical events to the aggregate root", func() {
		By("recording an event via a different executor", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
			}

			e := &CommandExecutor{
				HandlerIdentity: &envelopespec.Identity{
					Name: "<handler-name>",
					Key:  "<handler-key>",
				},
				Handler:       handler,
				Packer:        packer,
				JournalOpener: journalOpener,
				EventAppender: events,
				Logger:        zapx.NewTesting("<handler-name>"),
			}

			run(
				ctx,
				e,
				func(ctx context.Context) error {
					return e.ExecuteCommand(
						ctx,
						"<instance-id>",
						packer.Pack(MessageC1),
					)
				},
			)
		})

		By("asserting that the event is applied to the aggregate root in the original executor", func() {
			handler.HandleCommandFunc = func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				defer GinkgoRecover()

				stub := r.(*AggregateRoot)
				Expect(stub.AppliedEvents).To(ConsistOf(MessageE1))
			}

			run(
				ctx,
				executor,
				func(ctx context.Context) error {
					return executor.ExecuteCommand(
						ctx,
						"<instance-id>",
						packer.Pack(MessageC1),
					)
				},
			)
		})
	})

	It("keeps the aggregate state across requests", func() {
		handler.HandleCommandFunc = func(
			r dogma.AggregateRoot,
			s dogma.AggregateCommandScope,
			m dogma.Message,
		) {
			stub := r.(*AggregateRoot)
			count := len(stub.AppliedEvents)

			s.RecordEvent(MessageE{
				Value: count,
			})
		}

		run(
			ctx,
			executor,
			func(ctx context.Context) error {
				if err := executor.ExecuteCommand(
					ctx,
					"<instance-id>",
					packer.Pack(MessageC1),
				); err != nil {
					return err
				}

				if err := executor.ExecuteCommand(
					ctx,
					"<instance-id>",
					packer.Pack(MessageC1),
				); err != nil {
					return err
				}

				return nil
			},
		)

		err := eventstreamtest.ContainsEvents(
			ctx,
			events,
			fixtures.Marshaler,
			// use floats because messages are marshaled to JSON
			MessageE{Value: float64(0)},
			MessageE{Value: float64(1)},
		)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func run(
	ctx context.Context,
	e *CommandExecutor,
	fn func(ctx context.Context) error,
) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var g sync.WaitGroup

	g.Add(1)
	go func() {
		defer g.Done()
		defer GinkgoRecover()

		err := fn(ctx)
		Expect(err).ShouldNot(HaveOccurred(), "fn() should not return an error")

		cancel()
	}()

	g.Add(1)
	go func() {
		defer g.Done()
		defer GinkgoRecover()

		err := e.Run(ctx)
		Expect(err).To(Equal(context.Canceled), "Run() should exit due to context cancelation")
	}()

	g.Wait()
}
