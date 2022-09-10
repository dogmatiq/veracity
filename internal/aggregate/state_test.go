package aggregate_test

import (
	"context"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/aggregate"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
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

			cancel()
		}

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			return executor.ExecuteCommand(
				ctx,
				"<instance-id>",
				packer.Pack(MessageC1),
			)
		})

		g.Go(func() error {
			return executor.Run(ctx)
		})

		err := g.Wait()
		Expect(err).To(Equal(context.Canceled))
	})
})
