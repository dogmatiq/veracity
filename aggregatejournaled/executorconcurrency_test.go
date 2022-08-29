package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/aggregatejournaled"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type CommandExecutor (concurrency)", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		journal *journalStub
		stream  *eventStreamStub
		handler *AggregateMessageHandler
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
				s.RecordEvent(MessageE{
					Value: m.(MessageC).Value,
				})
			},
		}
	})

	Describe("func ExecuteCommand()", func() {
		It("writes the events to the stream exactly once", func() {
			packer := NewPacker(
				message.TypeRoles{
					MessageCType: message.CommandRole,
					MessageEType: message.EventRole,
				},
			)

			const eventCount = 100
			var expect []string
			queue := make(chan parcel.Parcel, eventCount)

			for i := 0; i < eventCount; i++ {
				v := fmt.Sprintf("<message-%d>", i)
				expect = append(expect, v)

				queue <- packer.PackCommand(
					MessageC{Value: v},
				)
			}

			g, ctx := errgroup.WithContext(ctx)

			for i := 0; i < 10; i++ {
				g.Go(func() error {
				reload:
					for {
						executor := &CommandExecutor{
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
						if err == ErrConflict {
							continue reload
						}
						if err != nil {
							return err
						}

						for {
							acked := false
							ack := func(ctx context.Context) error {
								acked = true
								return nil
							}

							var command parcel.Parcel
							select {
							default:
								return nil
							case command = <-queue:
							}

							err = executor.ExecuteCommand(ctx, command, ack)
							if !acked {
								queue <- command
							}
							if err == ErrConflict {
								continue reload
							}
							if err != nil {
								return err
							}

							if !acked {
								return errors.New("executor succeeded without acknowledging the command")
							}
						}
					}
				})
			}

			err := g.Wait()
			Expect(err).ShouldNot(HaveOccurred())

			var actual []any
			for _, p := range readAllEvents(ctx, stream, 0) {
				actual = append(actual, p.Message.(MessageE).Value)
			}

			Expect(actual).To(ConsistOf(expect))
		})
	})
})
