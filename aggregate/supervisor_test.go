package aggregate_test

import (
	"context"
	"errors"
	"time"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Supervisor", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		commands    chan *Command
		handler     *AggregateMessageHandler
		supervisor  *Supervisor
		startWorker func(
			ctx context.Context,
			id string,
			idle chan<- string,
			done func(error),
		) chan<- *Command
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

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

		startWorker = func(
			ctx context.Context,
			id string,
			idle chan<- string,
			done func(error),
		) chan<- *Command {
			panic("not implemented")
		}

		supervisor = &Supervisor{
			Handler:  handler,
			Commands: commands,
			StartWorker: func(
				ctx context.Context,
				id string,
				idle chan<- string,
				done func(error),
			) chan<- *Command {
				return startWorker(ctx, id, idle, done)
			},
			Logger: logging.DefaultLogger,
		}
	})

	Describe("func Run()", func() {
		When("waiting for a command", func() {
			It("returns an error if the context is canceled", func() {
				cancel()
				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.Canceled))
			})

			It("returns an error if a worker fails", func() {
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					go done(errors.New("<error>"))
					return nil
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
				Expect(err).To(MatchError("<error>"))
			})

			It("shuts down idle workers", func() {
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					commands := make(chan *Command)

					go func() {
						// Handle the command that caused this worker to start.
						select {
						case <-ctx.Done():
							done(ctx.Err())
						case cmd, ok := <-commands:
							if ok {
								cmd.Ack()
							}
						}

						// Wait for supervisor to receive the worker's idle
						// notification.
						select {
						case <-ctx.Done():
							done(ctx.Err())
						case idle <- id:
						}

						// Wait for permission to shutdown.
						select {
						case <-ctx.Done():
							done(ctx.Err())
						case _, ok := <-commands:
							if ok {
								done(errors.New("unexpected command"))
								return
							}
						}

						// Confirm that the supervisor accepts another idle
						// notification, even if it's already closed the command
						// channel.
						select {
						case <-ctx.Done():
							done(ctx.Err())
						case idle <- id:
							cancel()
							done(nil)
						}
					}()

					return commands
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
				Expect(err).To(Equal(context.Canceled))
			})

			It("restarts a worker that has shutdown if another command is routed to the same instance", func() {
				restarting := false
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					if restarting {
						go done(errors.New("<error-on-restart>"))
					}

					go done(nil)
					restarting = true

					return nil
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
				Expect(err).To(MatchError("<error-on-restart>"))
			})
		})

		When("waiting for a worker to accept a command", func() {
			It("returns an error if the context is canceled", func() {
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					go func() {
						<-ctx.Done()
						done(nil)
					}()

					cancel()

					return make(chan<- *Command)
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
				Expect(err).To(Equal(context.Canceled))
			})

			It("does not return if the command's context is canceled", func() {
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					go func() {
						<-ctx.Done()
						done(nil)
					}()

					return make(chan<- *Command)
				}

				go func() {
					defer GinkgoRecover()

					cmdCtx, cancelCommand := context.WithCancel(context.Background())

					cmd := executeCommandAsync(
						cmdCtx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					cancelCommand()

					Eventually(cmd.Done()).Should(BeClosed())
					Expect(cmd.Err()).To(Equal(context.Canceled))
				}()

				// Wait for test timeout to prove that the supervisor
				// is not shutting down as a result of the command's
				// context being canceled.
				err := supervisor.Run(ctx)
				Expect(err).To(Equal(context.DeadlineExceeded))
			})

			It("returns an error if the worker fails", func() {
				startWorker = func(
					ctx context.Context,
					id string,
					idle chan<- string,
					done func(error),
				) chan<- *Command {
					go done(errors.New("<error>"))
					return make(chan<- *Command)
				}

				go func() {
					defer GinkgoRecover()

					cmd := executeCommandAsync(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					Eventually(cmd.Done()).Should(BeClosed())
					Expect(cmd.Err()).To(MatchError("shutting down"))
				}()

				err := supervisor.Run(ctx)
				Expect(err).To(MatchError("<error>"))
			})

			When("when a worker becomes idle", func() {
				When("the worker belongs to the aggregate instance targetted by the command", func() {
					It("does not shutdown the worker", func() {
						startWorker = func(
							ctx context.Context,
							id string,
							idle chan<- string,
							done func(error),
						) chan<- *Command {
							commands := make(chan *Command)

							go func() {
								// Wait for supervisor to receive the worker's
								// idle notification.
								select {
								case <-ctx.Done():
									done(ctx.Err())
								case idle <- id:
								}

								// Expect commands NOT to be closed, instead we
								// should immediately receive the that the
								// supervisor is trying to dispatch.
								select {
								case <-ctx.Done():
									done(ctx.Err())
								case cmd, ok := <-commands:
									if ok {
										cmd.Ack()
									}
								}

								<-ctx.Done()
								done(ctx.Err())
							}()

							return commands
						}

						go func() {
							defer GinkgoRecover()

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

					It("restarts the worker if it shuts down", func() {
						first := true
						startWorker = func(
							ctx context.Context,
							id string,
							idle chan<- string,
							done func(error),
						) chan<- *Command {
							commands := make(chan *Command)

							if first {
								// On the first run we shutdown the worker
								// immediately.
								go done(nil)
								first = false
							} else {
								// Otherwise we tell the supervisor to shutdown.
								go func() {
									cancel()
									done(nil)
								}()
							}

							return commands
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
						Expect(err).To(Equal(context.Canceled))
					})
				})

				When("the worker does NOT belong to the aggregate instance targetted by the command", func() {
					barrier := make(chan struct{})

					It("shuts the worker down", func() {
						startWorker = func(
							ctx context.Context,
							id string,
							idle chan<- string,
							done func(error),
						) chan<- *Command {
							commands := make(chan *Command)

							// The worker for "<instance-1>" blocks until the
							// supervisor is shutdown. This keeps the supervisor
							// in its "dispatch" state.
							if id == "<instance-1>" {
								go func() {
									<-ctx.Done()
									done(ctx.Err())
								}()

								// Notify the worker for "<instance-2>" that the
								// supervisor is blocked so it may enter the
								// idle state.
								barrier <- struct{}{}

								return commands
							}

							// The worker for "<instance-2>" handles its command
							// then enters the idle state.
							go func() {
								// Handle the command that caused this worker to
								// start.
								select {
								case <-ctx.Done():
									done(ctx.Err())
								case cmd, ok := <-commands:
									if ok {
										cmd.Ack()
									}
								}

								// Wait to enter the idle state until the worker
								// for <instance-1> has started, and therefore
								// the supervisor is in its "dispatch" state.
								<-barrier

								// Wait for supervisor to receive the worker's
								// idle notification.
								select {
								case <-ctx.Done():
									done(ctx.Err())
								case idle <- id:
								}

								// Wait for permission to shutdown.
								select {
								case <-ctx.Done():
									done(ctx.Err())
								case _, ok := <-commands:
									if ok {
										done(errors.New("unexpected command"))
									} else {
										cancel()
										done(nil)
									}
								}
							}()

							return commands
						}

						go func() {
							defer GinkgoRecover()

							// Send a command to <instance-2>, wait for it to
							// complete, then start the worker for <instance-1>.
							executeCommandSync(
								ctx,
								commands,
								NewParcel("<command-2>", MessageC2),
							)

							executeCommandAsync(
								ctx,
								commands,
								NewParcel("<command-1>", MessageC1),
							)
						}()

						err := supervisor.Run(ctx)
						Expect(err).To(Equal(context.Canceled))
					})
				})
			})
		})

		// XIt("waits for all workers to finish when the context is canceled", func() {

		// })
	})
})
