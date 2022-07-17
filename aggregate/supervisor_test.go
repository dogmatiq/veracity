package aggregate_test

import (
	"context"
	"errors"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Supervisor", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		eventStore  *memory.AggregateEventStore
		eventReader *eventReaderStub
		eventWriter *eventWriterStub

		packer     *parcel.Packer
		loader     *Loader
		commands   chan *Command
		handler    *AggregateMessageHandler
		logger     *logging.BufferedLogger
		supervisor *Supervisor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		DeferCleanup(cancel)

		eventStore = &memory.AggregateEventStore{}

		eventReader = &eventReaderStub{
			EventReader: eventStore,
		}

		eventWriter = &eventWriterStub{
			EventWriter: eventStore,
		}

		loader = &Loader{
			EventReader: eventReader,
			Marshaler:   Marshaler,
		}

		commands = make(chan *Command)

		handler = &AggregateMessageHandler{
			ConfigureFunc: func(c dogma.AggregateConfigurer) {
				c.Identity("<handler-name>", "<handler-key>")
				c.ConsumesCommandType(MessageC{})
				c.ProducesEventType(MessageE{})
			},
			RouteCommandToInstanceFunc: func(m dogma.Message) string {
				return "<instance>"
			},
		}

		packer = NewPacker(
			message.TypeRoles{
				MessageCType: message.CommandRole,
				MessageEType: message.EventRole,
			},
		)

		logger = &logging.BufferedLogger{}

		supervisor = &Supervisor{
			WorkerConfig: WorkerConfig{
				Handler:         handler,
				HandlerIdentity: configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Packer:          packer,
				Loader:          loader,
				EventWriter:     eventWriter,
				Logger:          logger,
			},
			Commands:      commands,
			CommandBuffer: 1,
		}
	})

	Describe("func Run()", func() {
		When("waiting for a command", func() {
			It("returns an error if the context is canceled", func() {
				cancel()
				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})

			It("returns an error if a worker returns an error", func() {
				eventReader.ReadBoundsFunc = func(
					ctx context.Context,
					hk, id string,
				) (uint64, uint64, error) {
					return 0, 0, errors.New("<error>")
				}

				go func() {
					defer GinkgoRecover()

					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(
					MatchError(
						"aggregate root <handler-name>[<instance>] cannot be loaded: unable to read event revision bounds: <error>",
					),
				)
			})

			It("restarts a worker that has shutdown if another command is routed to the same instance", func() {
				supervisor.WorkerConfig.IdleTimeout = 10 * time.Millisecond

				go func() {
					defer GinkgoRecover()

					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					time.Sleep(20 * time.Millisecond)

					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					cancel()
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})
		})

		When("waiting for a worker to accept a command", func() {
			It("returns an error if the context is canceled", func() {
				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					<-ctx.Done()
				}

				var cmd *Command
				go func() {
					defer GinkgoRecover()

					// Execute the command that causes the worker to block
					// waiting for the context.
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					// Fill up the worker's command buffer (size == 1).
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC2),
					)

					// Cause the supervisor to block waiting for the worker's
					// command channel to become unblocked.
					//
					// Note that we don't use ctx, otherwise it will report the
					// context cancelation error instead of the supervisor
					// shutdown error.
					cmd = executeCommandAsync(
						context.Background(),
						commands,
						NewParcel("<command>", MessageC3),
					)

					cancel()
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))

				select {
				case <-time.After(1 * time.Second):
					Fail("timed-out waiting for result")
				case <-cmd.Done():
					Expect(cmd.Err()).To(
						MatchError(
							"shutting down",
						),
					)
				}
			})

			XIt("returns an error if the command's context is canceled", func() {
			})

			XIt("returns an error if a worker fails", func() {
			})

			XIt("does not shutdown the destination worker if it becomes idle", func() {
			})

			XIt("restarts the destination worker if it shuts down", func() {
			})
		})

		XIt("waits for all workers to finish when the context is canceled", func() {
		})

		XIt("uses the a single worker for a given aggregate instance", func() {
		})
	})
})
