package aggregate_test

import (
	"context"
	"errors"
	"sync"
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
	"golang.org/x/sync/errgroup"
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
		observer   *observerStub
		supervisor *Supervisor
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
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
			Logger:      logging.SilentLogger,
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

		observer = &observerStub{}

		supervisor = &Supervisor{
			WorkerConfig: WorkerConfig{
				Handler:         handler,
				HandlerIdentity: configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Packer:          packer,
				Loader:          loader,
				EventWriter:     eventWriter,
				Observer:        observer,
				Logger:          logging.DefaultLogger,
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

				g, ctx := errgroup.WithContext(ctx)
				defer g.Wait()

				g.Go(func() error {
					return supervisor.Run(ctx)
				})

				g.Go(func() error {
					defer GinkgoRecover()

					executeCommandAsync(
						ctx,
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					return nil
				})

				err := g.Wait()
				Expect(err).To(
					MatchError(
						"aggregate root <handler-name>[<instance-1>] cannot be loaded: unable to read event revision bounds: <error>",
					),
				)
			})

			It("restarts a worker that has shutdown if another command is routed to the same instance", func() {
				supervisor.IdleTimeout = 5 * time.Millisecond

				var once sync.Once
				barrier := make(chan struct{})
				observer.OnWorkerShutdownFunc = func(
					h configkit.Identity,
					id string,
					err error,
				) {
					once.Do(func() {
						openBarrier(barrier)
					})
				}

				g, ctx := errgroup.WithContext(ctx)
				defer g.Wait()

				g.Go(func() error {
					return supervisor.Run(ctx)
				})

				g.Go(func() error {
					defer GinkgoRecover()

					executeCommandSync(
						ctx,
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					// Wait for the worker to shutdown.
					waitAtBarrier(barrier)

					// Verify that a new worker is started by ensuring this
					// command is actually handled.
					executeCommandSync(
						ctx,
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					cancel()

					return nil
				})

				err := g.Wait()
				Expect(err).To(Equal(context.Canceled))
			})
		})

		When("waiting for a worker to accept a command", func() {
			var (
				waitGroup          *errgroup.Group
				workerStartBarrier chan struct{}
			)

			BeforeEach(func() {
				waitGroup = &errgroup.Group{}
				DeferCleanup(func() {
					waitGroup.Wait()
				})
			})

			JustBeforeEach(func() {
				barrier := make(chan struct{})
				{
					var once sync.Once
					observer.OnCommandDispatchedFunc = func(
						h configkit.Identity,
						id string,
						cmd *Command,
					) {
						if id == "<instance-1>" {
							// Allow the test to proceed once we've filled the
							// worker's command channel.
							once.Do(func() {
								openBarrier(barrier)
							})
						}
					}
				}

				workerStartBarrier = make(chan struct{})
				{
					var once sync.Once
					observer.OnWorkerStartFunc = func(
						h configkit.Identity,
						id string,
					) {
						if id == "<instance-1>" {
							// Wait for permission to proceed to handle the command.
							// At this stage it's still on the commands channel.
							once.Do(func() {
								waitAtBarrier(workerStartBarrier)
							})
						}
					}
				}

				// Start the supervisor in the background.
				waitGroup.Go(func() error {
					defer GinkgoRecover()
					return supervisor.Run(ctx)
				})

				// Send a command both to cause the worker to start and to
				// fill its command channel to capacity (buffer size = 1).
				executeCommandAsync(
					ctx,
					context.Background(),
					commands,
					NewParcel("<accepted-command>", MessageC1),
				)

				// Don't let the test start until the worker's command channel
				// is filled.
				waitAtBarrier(barrier)
			})

			// sendBlockedCommand sends another command to put the supervisor
			// into the "dispatch" state.
			putSupervisorIntoDispatchState := func() (*Command, context.CancelFunc) {
				cmdCtx, cancelBlockedCommand := context.WithCancel(context.Background())

				return executeCommandAsync(
					ctx,
					cmdCtx,
					commands,
					NewParcel("<blocked-command>", MessageC1),
				), cancelBlockedCommand
			}

			When("the supervisor's context is canceled", func() {
				It("returns an error", func() {
					blockedCommand, cancelBlockedCommand := putSupervisorIntoDispatchState()
					defer cancelBlockedCommand()

					cancel()

					Eventually(blockedCommand.Done()).Should(BeClosed())
					Expect(blockedCommand.Err()).To(MatchError("shutting down"))

					openBarrier(workerStartBarrier)

					err := waitGroup.Wait()
					Expect(err).To(Equal(context.Canceled))
				})
			})

			When("the command's context is canceled", func() {
				It("does not stop the supervisor", func() {
					blockedCommand, cancelBlockedCommand := putSupervisorIntoDispatchState()

					cancelBlockedCommand()
					Eventually(blockedCommand.Done()).Should(BeClosed())
					Expect(blockedCommand.Err()).To(Equal(context.Canceled))

					openBarrier(workerStartBarrier)

					// Wait for test timeout to prove that the supervisor
					// is not shutting down as a result of the command's
					// context being canceled.
					err := waitGroup.Wait()
					Expect(err).To(Equal(context.DeadlineExceeded))
				})
			})

			When("when the worker fails", func() {
				BeforeEach(func() {
					eventReader.ReadBoundsFunc = func(
						ctx context.Context,
						hk, id string,
					) (uint64, uint64, error) {
						return 0, 0, errors.New("<error>")
					}
				})

				It("it returns an error", func() {
					blockedCommand, cancelBlockedCommand := putSupervisorIntoDispatchState()
					defer cancelBlockedCommand()

					openBarrier(workerStartBarrier)

					err := waitGroup.Wait()
					Expect(err).To(
						MatchError(
							"aggregate root <handler-name>[<instance-1>] cannot be loaded: unable to read event revision bounds: <error>",
						),
					)

					Eventually(blockedCommand.Done()).Should(BeClosed())
					Expect(blockedCommand.Err()).To(MatchError("shutting down"))
				})
			})

			When("when a worker becomes idle", func() {
				When("the worker belongs to the aggregate instance targetted by the command", func() {
					var handleBarrier, receiveBarrier, shutdownBarrier chan struct{}

					BeforeEach(func() {
						handleBarrier = make(chan struct{})
						receiveBarrier = make(chan struct{})
						shutdownBarrier = make(chan struct{})

						handler.HandleCommandFunc = func(
							r dogma.AggregateRoot,
							s dogma.AggregateCommandScope,
							m dogma.Message,
						) {
							waitAtBarrier(handleBarrier)

							// Destroy the instance, causing the worker to enter
							// an idle state.
							s.RecordEvent(MessageE1)
							s.Destroy()
						}

						observer.OnCommandReceivedFunc = func(
							h configkit.Identity,
							id string,
							cmd *Command,
						) {
							if cmd.Parcel.Envelope.GetMessageId() == "<additional-blocked-command>" {
								waitAtBarrier(receiveBarrier)
							}
						}

						observer.OnWorkerShutdownFunc = func(
							h configkit.Identity,
							id string,
							err error,
						) {
							waitAtBarrier(shutdownBarrier)
						}
					})

					It("does not shutdown the worker", func() {
						blockedCommand, cancelBlockedCommand := putSupervisorIntoDispatchState()
						defer cancelBlockedCommand()

						// Let the worker receive the first command from the
						// channel.
						//
						// The "blockedCommand" is no longer blocked, it is now
						// in the worker's command channel, filling it to
						// capacity (size == 1).
						openBarrier(workerStartBarrier)

						// Enqueue another command that will become blocked.
						additionalBlockedCommand := executeCommandAsync(
							ctx,
							ctx,
							commands,
							NewParcel("<additional-blocked-command>", MessageC1),
						)

						// Wait for this additional blocked command to be
						// received by the worker.
						openBarrier(receiveBarrier)

						// Allow the handler to complete the very first command
						// which will cause the handler to destroy the instance.
						openBarrier(handleBarrier)

						// Allow the original blocked command to complete.
						openBarrier(handleBarrier)
						Eventually(blockedCommand.Done()).Should(BeClosed())
						Expect(blockedCommand.Err()).ShouldNot(HaveOccurred())

						// Allow the additional blocked command to complete.
						openBarrier(handleBarrier)
						Eventually(additionalBlockedCommand.Done()).Should(BeClosed())
						Expect(additionalBlockedCommand.Err()).ShouldNot(HaveOccurred())

						// Only let the worker shutdown _after_ the blocked
						// commands results are made available.
						openBarrier(shutdownBarrier)

						cancel()
					})
				})

				When("the worker does NOT belong to the aggregate instance targetted by the command", func() {
					var shutdownBarrier chan struct{}

					BeforeEach(func() {
						supervisor.IdleTimeout = 5 * time.Millisecond

						shutdownBarrier = make(chan struct{})

						observer.OnWorkerShutdownFunc = func(
							h configkit.Identity,
							id string,
							err error,
						) {
							if id == "<instance-2>" {
								openBarrier(shutdownBarrier)
							}
						}
					})

					It("shuts the worker down", func() {
						// Send a command that is routed to a different instance
						// and hence a different worker.
						executeCommandAsync(
							ctx,
							ctx,
							commands,
							NewParcel("<command-for-other-worker>", MessageC2),
						)

						_, cancelBlockedCommand := putSupervisorIntoDispatchState()
						defer cancelBlockedCommand()

						// Wait for the worker for <instance-2> to shutdown.
						waitAtBarrier(shutdownBarrier)

						cancel()

						// Let the worker for <instance-1> continue/shutdown.
						openBarrier(workerStartBarrier)
					})
				})
			})

			// 	XIt("restarts the destination worker if it shuts down", func() {
			// 		barrier := make(chan struct{})

			// 		handler.HandleCommandFunc = func(
			// 			r dogma.AggregateRoot,
			// 			s dogma.AggregateCommandScope,
			// 			m dogma.Message,
			// 		) {
			// 			s.RecordEvent(MessageE1)
			// 			s.Destroy()
			// 		}

			// 		supervisor.SetIdleTestHook(
			// 			func(id string) {
			// 				select {
			// 				case barrier <- struct{}{}:
			// 				case <-ctx.Done():
			// 					Expect(ctx.Err()).ShouldNot(HaveOccurred())
			// 				}
			// 				supervisor.SetIdleTestHook(nil)
			// 			},
			// 		)

			// 		var cmd *Command
			// 		go func() {
			// 			defer GinkgoRecover()

			// 			// Send a command to cause destruction of the instance.
			// 			executeCommandSync(
			// 				ctx,
			// 				commands,
			// 				NewParcel("<command>", MessageC1),
			// 			)

			// 			select {
			// 			case <-barrier:
			// 			case <-ctx.Done():
			// 				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			// 			}

			// 			// Fill up the worker's command buffer (size == 1).
			// 			executeCommandAsync(
			// 				ctx,
			// 				commands,
			// 				NewParcel("<command>", MessageC2),
			// 			)

			// 			// Cause the supervisor to block waiting for the worker's
			// 			// command channel to become unblocked.
			// 			cmd = executeCommandAsync(
			// 				ctx,
			// 				commands,
			// 				NewParcel("<command>", MessageC3),
			// 			)

			// 			select {
			// 			case <-time.After(1 * time.Second):
			// 				Fail("timed-out waiting for result")
			// 			case <-cmd.Done():
			// 				cancel()
			// 			}
			// 		}()

			// 		err := supervisor.Run(ctx)
			// 		Expect(err).To(Equal(context.Canceled))
			// 		Expect(cmd.Err()).ShouldNot(HaveOccurred())
			// 	})
		})

		// XIt("waits for all workers to finish when the context is canceled", func() {

		// })
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
