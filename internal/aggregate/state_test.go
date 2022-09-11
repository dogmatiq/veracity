package aggregate_test

import (
	"context"
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
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type CommandExecutor (aggregate root state)", func() {
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		packer   *envelope.Packer
		handler  *AggregateMessageHandler
		events   *eventstream.EventStream
		executor *CommandExecutor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()

		handler = &AggregateMessageHandler{}

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
			JournalOpener: &memory.JournalOpener[*JournalRecord]{},
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

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer GinkgoRecover()

		if err := fn(ctx); err != nil {
			return err
		}

		cancel()

		return nil
	})

	g.Go(func() error {
		defer GinkgoRecover()

		return e.Run(ctx)
	})

	err := g.Wait()
	Expect(err).To(Equal(context.Canceled))
	Expect(ctx.Err()).To(Equal(context.Canceled))
}
