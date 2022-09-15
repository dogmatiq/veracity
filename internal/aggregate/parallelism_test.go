package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/internal/aggregate"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type CommandExecutor (parallelism)", func() {
	It("handles each command exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		packer := envelope.NewTestPacker()
		journals := &memory.JournalStore[*JournalRecord]{}
		eventJournal := memory.NewJournal[[]byte]()

		var (
			parallelism = runtime.NumCPU()
			messages    = parallelism * 500
			instances   = 10
		)

		var expect []dogma.Message
		var envelopes []*envelopespec.Envelope

		for i := 0; i < messages; i++ {
			instanceID := fmt.Sprintf("<instance-%d>", i%instances)
			value := fmt.Sprintf("<value-%d>", i)

			envelopes = append(
				envelopes,
				packer.Pack(
					MessageC{
						Value: value,
					},
				),
			)

			expect = append(
				expect,
				MessageE{
					// The handler produces events with a value of the aggregate
					// instance ID concatenated with the command's value.
					//
					// This lets us assert that the commands are being routed to
					// the correct instance.
					Value: fmt.Sprintf("%s-%s", instanceID, value),
				},
			)
		}

		tick := func(ctx context.Context) error {
			exec := &CommandExecutor{
				HandlerIdentity: &envelopespec.Identity{
					Name: "<handler-name>",
					Key:  "<handler-key>",
				},
				Handler: &AggregateMessageHandler{
					HandleCommandFunc: func(
						r dogma.AggregateRoot,
						s dogma.AggregateCommandScope,
						m dogma.Message,
					) {
						s.RecordEvent(MessageE{
							Value: fmt.Sprintf(
								"%s-%s",
								s.InstanceID(),
								m.(MessageC).Value,
							),
						})
					},
				},
				Packer:       packer,
				JournalStore: journals,
				EventAppender: &eventstream.EventStream{
					Journal: eventJournal,
					Logger:  zap.NewNop(),
				},
				Logger: zap.NewNop(),
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			g, ctx := errgroup.WithContext(ctx)

			g.Go(func() error {
				err := exec.Run(ctx)
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			})

			g.Go(func() error {
				for i, env := range envelopes {
					id := fmt.Sprintf("<instance-%d>", i%instances)
					if err := exec.ExecuteCommand(ctx, id, env); err != nil {
						return err
					}
				}

				cancel()
				return nil
			})

			return g.Wait()
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					err := tick(ctx)
					if err == nil || errors.Is(err, context.DeadlineExceeded) {
						return nil
					}
				}
			})
		}

		err := g.Wait()
		Expect(err).ShouldNot(HaveOccurred())

		events := &eventstream.EventStream{
			Journal: eventJournal,
			Logger:  zap.NewNop(),
		}

		var actual []dogma.Message
		err = events.Range(
			ctx,
			0,
			func(
				ctx context.Context,
				env *envelopespec.Envelope,
			) (bool, error) {
				m, err := marshalkit.UnmarshalMessageFromEnvelope(Marshaler, env)
				Expect(err).ShouldNot(HaveOccurred())
				actual = append(actual, m)
				return true, nil
			},
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})
})
