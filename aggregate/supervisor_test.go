package aggregate_test

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Supervisor", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		waitGroup errgroup.Group

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
				switch m {
				case MessageC1:
					return "<instance-1>"
				case MessageC2:
					return "<instance-2>"
				case MessageC3:
					return "<instance-3>"
				default:
					panic("unexpected message")
				}
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

	ReportAfterEach(func(SpecReport) {
		waitGroup.Wait()
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

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					return nil
				})

				err := supervisor.Run(ctx)
				Expect(err).To(
					MatchError(
						"aggregate root <handler-name>[<instance-1>] cannot be loaded: unable to read event revision bounds: <error>",
					),
				)
			})

			It("restarts a worker that has shutdown if another command is routed to the same instance", func() {
				supervisor.WorkerConfig.IdleTimeout = 10 * time.Millisecond
				barrier := make(chan struct{})

				supervisor.SetShutdownTestHook(
					func(id string, err error) {
						By("notifying the test that the worker has finished")
						openBarrier(barrier)
						supervisor.SetShutdownTestHook(nil)
					},
				)

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					By("waiting for the worker to finish")
					waitAtBarrier(barrier)

					// Verify that a new worker is started by ensuring this
					// command is actually handled.
					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					By("canceling the supervisor's context")
					cancel()

					return nil
				})

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})
		})

		When("waiting for a worker to accept a command", func() {
			var workerBarrier chan struct{}

			BeforeEach(func() {
				workerBarrier = make(chan struct{})

				var count uint32
				eventReader.ReadBoundsFunc = func(
					ctx context.Context,
					hk, id string,
				) (uint64, uint64, error) {
					defer GinkgoRecover()

					if atomic.AddUint32(&count, 1) == 1 {
						By("notifying the test that the worker has started")
						openBarrier(workerBarrier)

						By("stopping the worker until the test wants to proceed")
						waitAtBarrier(workerBarrier)
					}

					return eventStore.ReadBounds(ctx, hk, id)
				}

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					By("filling the worker's command channel to capacity")
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					return nil
				})
			})

			It("returns an error if the context is canceled", func() {
				var cmd *Command

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					By("waiting for the worker to start")
					waitAtBarrier(workerBarrier)

					By("putting the supervisor into the 'dispatch' state")
					cmd = executeCommandAsync(
						context.Background(),
						commands,
						NewParcel("<command>", MessageC1),
					)

					By("canceling the supervisor's context")
					cancel()

					By("letting the worker finish")
					openBarrier(workerBarrier)

					return nil
				})

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))

				Eventually(cmd.Done(), "500ms").Should(BeClosed())
				Expect(cmd.Err()).To(
					MatchError(
						"shutting down",
					),
				)
			})

			It("returns an error if the command's context is canceled", func() {
				var cmd *Command

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					By("waiting for the worker to start")
					waitAtBarrier(workerBarrier)

					By("putting the supervisor into the 'dispatch' state")
					cmdCtx, cmdCancel := context.WithCancel(ctx)
					defer cmdCancel()

					cmd = executeCommandAsync(
						cmdCtx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					By("canceling the command's context")
					cmdCancel()

					By("waiting for the command's done channel to close")
					Eventually(cmd.Done(), "500ms").Should(BeClosed())

					By("canceling the supervisor's context")
					cancel()

					By("letting the worker finish")
					openBarrier(workerBarrier)

					return nil
				})

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
				Expect(cmd.Err()).To(Equal(context.Canceled))
			})

			It("returns an error if a worker fails", func() {
				var cmd *Command

				fn := eventReader.ReadBoundsFunc
				eventReader.ReadBoundsFunc = func(
					ctx context.Context,
					hk, id string,
				) (uint64, uint64, error) {
					fn(ctx, hk, id)
					return 0, 0, errors.New("<error>")
				}

				waitGroup.Go(func() error {
					defer GinkgoRecover()

					By("waiting for the worker to start")
					waitAtBarrier(workerBarrier)

					By("putting the supervisor into the 'dispatch' state")
					cmd = executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					By("letting the worker return an error")
					openBarrier(workerBarrier)

					return nil
				})

				err := supervisor.Run(ctx)
				Expect(err).To(
					MatchError(
						"aggregate root <handler-name>[<instance-1>] cannot be loaded: unable to read event revision bounds: <error>",
					),
				)

				Eventually(cmd.Done(), "500ms").Should(BeClosed())
				Expect(cmd.Err()).To(
					MatchError(
						"shutting down",
					),
				)
			})

			XIt("does not shutdown the destination worker if it becomes idle", func() {
				barrier := make(chan struct{})

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					s.RecordEvent(MessageE1)
					s.Destroy()
				}

				eventWriter.WriteEventsFunc = func(
					ctx context.Context,
					hk, id string,
					begin, end uint64,
					events []*envelopespec.Envelope,
				) error {
					select {
					case barrier <- struct{}{}:
					case <-ctx.Done():
						Expect(ctx.Err()).ShouldNot(HaveOccurred())
					}

					eventWriter.WriteEventsFunc = nil
					return eventStore.WriteEvents(ctx, hk, id, begin, end, events)
				}

				var cmd *Command
				go func() {
					defer GinkgoRecover()

					// Send a command to cause destruction of the instance.
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
					cmd = executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC3),
					)

					select {
					case <-barrier:
					case <-ctx.Done():
						Expect(ctx.Err()).ShouldNot(HaveOccurred())
					}

					select {
					case <-time.After(1 * time.Second):
						Fail("timed-out waiting for result")
					case <-cmd.Done():
						cancel()
					}
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
				Expect(cmd.Err()).ShouldNot(HaveOccurred())
			})

			XIt("shuts down other workers if they become idle", func() {
				barrier := make(chan struct{})

				handler.RouteCommandToInstanceFunc = func(m dogma.Message) string {
					if m == MessageC1 {
						return "<instance-1>"
					}

					return "<instance-2>"
				}

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					if s.InstanceID() == "<instance-2>" {
						s.RecordEvent(MessageE1)
						s.Destroy()
					}
				}

				eventWriter.WriteEventsFunc = func(
					ctx context.Context,
					hk, id string,
					begin, end uint64,
					events []*envelopespec.Envelope,
				) error {
					if id == "<instance-1>" {
						select {
						case barrier <- struct{}{}:
						case <-ctx.Done():
							Expect(ctx.Err()).ShouldNot(HaveOccurred())
						}
					}

					eventWriter.WriteEventsFunc = nil
					return eventStore.WriteEvents(ctx, hk, id, begin, end, events)
				}

				var cmd *Command
				go func() {
					defer GinkgoRecover()

					// Fill up the worker's command buffer (size == 1).
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					// Cause the supervisor to block waiting for the worker's
					// command channel to become unblocked.
					cmd = executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC3),
					)

					// Send a command to cause destruction of the instance.
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					select {
					case <-barrier:
					case <-ctx.Done():
						Expect(ctx.Err()).ShouldNot(HaveOccurred())
					}

					select {
					case <-time.After(1 * time.Second):
						Fail("timed-out waiting for result")
					case <-cmd.Done():
						cancel()
					}
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
				Expect(cmd.Err()).ShouldNot(HaveOccurred())
			})

			XIt("restarts the destination worker if it shuts down", func() {
				barrier := make(chan struct{})

				handler.HandleCommandFunc = func(
					r dogma.AggregateRoot,
					s dogma.AggregateCommandScope,
					m dogma.Message,
				) {
					s.RecordEvent(MessageE1)
					s.Destroy()
				}

				supervisor.SetIdleTestHook(
					func(id string) {
						select {
						case barrier <- struct{}{}:
						case <-ctx.Done():
							Expect(ctx.Err()).ShouldNot(HaveOccurred())
						}
						supervisor.SetIdleTestHook(nil)
					},
				)

				var cmd *Command
				go func() {
					defer GinkgoRecover()

					// Send a command to cause destruction of the instance.
					executeCommandSync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					select {
					case <-barrier:
					case <-ctx.Done():
						Expect(ctx.Err()).ShouldNot(HaveOccurred())
					}

					// Fill up the worker's command buffer (size == 1).
					executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC2),
					)

					// Cause the supervisor to block waiting for the worker's
					// command channel to become unblocked.
					cmd = executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC3),
					)

					select {
					case <-time.After(1 * time.Second):
						Fail("timed-out waiting for result")
					case <-cmd.Done():
						cancel()
					}
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
				Expect(cmd.Err()).ShouldNot(HaveOccurred())
			})
		})

		XIt("waits for all workers to finish when the context is canceled", func() {

		})

		XIt("uses the a single worker for a given aggregate instance", func() {
		})
	})
})

func openBarrier(barrier chan<- struct{}) {
	select {
	case barrier <- struct{}{}:
	case <-time.After(5 * time.Second):
		panic("timed-out waiting for barrier")
	}
}

func waitAtBarrier(barrier <-chan struct{}) {
	select {
	case <-barrier:
	case <-time.After(5 * time.Second):
		panic("timed-out waiting for barrier")
	}
}
