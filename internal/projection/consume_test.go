package projection_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/projection"
	"github.com/dogmatiq/veracity/internal/test"
)

func TestConsume(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Packer        *envelope.Packer
		Handler       *ProjectionMessageHandler
		EventConsumer *eventConsumerStub
		Supervisor    *Supervisor
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Packer = newPacker()
		deps.Handler = &ProjectionMessageHandler{}
		deps.EventConsumer = &eventConsumerStub{}

		deps.Supervisor = &Supervisor{
			Handler:       deps.Handler,
			EventConsumer: deps.EventConsumer,
			Packer:        deps.Packer,
		}

		return deps
	}

	t.Run("it applies events exactly once, in order regardless of errors", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			Desc          string
			InduceFailure func(*dependencies)
		}{
			{
				Desc: "no faults",
				InduceFailure: func(*dependencies) {
				},
			},
			{
				Desc: "failure before handling event at offset 0",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleEventFunc

					deps.Handler.HandleEventFunc = func(
						ctx context.Context,
						r, c, n []byte,
						s dogma.ProjectionEventScope,
						e dogma.Event,
					) (bool, error) {
						if e == MessageE1 && done.CompareAndSwap(false, true) {
							return false, errors.New("<error>")
						}

						return handle(ctx, r, c, n, s, e)
					}
				},
			},
			{
				Desc: "failure after handling event at offset 0",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleEventFunc

					deps.Handler.HandleEventFunc = func(
						ctx context.Context,
						r, c, n []byte,
						s dogma.ProjectionEventScope,
						e dogma.Event,
					) (bool, error) {
						ok, err := handle(ctx, r, c, n, s, e)
						if !ok || err != nil {
							return ok, err
						}
						if e == MessageE1 && done.CompareAndSwap(false, true) {
							return false, errors.New("<error>")
						}

						return true, nil
					}
				},
			},
			{
				Desc: "failure before handling event at offset 1",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleEventFunc

					deps.Handler.HandleEventFunc = func(
						ctx context.Context,
						r, c, n []byte,
						s dogma.ProjectionEventScope,
						e dogma.Event,
					) (bool, error) {
						if e == MessageE2 && done.CompareAndSwap(false, true) {
							return false, errors.New("<error>")
						}

						return handle(ctx, r, c, n, s, e)
					}
				},
			},
			{
				Desc: "failure after handling event at offset 1",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleEventFunc

					deps.Handler.HandleEventFunc = func(
						ctx context.Context,
						r, c, n []byte,
						s dogma.ProjectionEventScope,
						e dogma.Event,
					) (bool, error) {
						ok, err := handle(ctx, r, c, n, s, e)
						if !ok || err != nil {
							return ok, err
						}
						if e == MessageE2 && done.CompareAndSwap(false, true) {
							return false, errors.New("<error>")
						}

						return true, nil
					}
				},
			},
			{
				Desc: "occ failure at offset 0",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					resourceVersionFunc := deps.Handler.ResourceVersionFunc

					deps.Handler.ResourceVersionFunc = func(ctx context.Context, r []byte) ([]byte, error) {
						if done.CompareAndSwap(false, true) {
							return []byte{0, 0, 0, 0, 0, 0, 0, 1}, nil
						}

						return resourceVersionFunc(ctx, r)
					}
				},
			},
			{
				Desc: "occ failure at offset 1",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleEventFunc

					deps.Handler.HandleEventFunc = func(
						ctx context.Context,
						r, c, n []byte,
						s dogma.ProjectionEventScope,
						e dogma.Event,
					) (bool, error) {
						if e == MessageE2 && done.CompareAndSwap(false, true) {
							return false, nil
						}

						return handle(ctx, r, c, n, s, e)
					}
				},
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.Desc, func(t *testing.T) {
				tctx := test.WithContext(t)

				deps := setup(tctx)

				var (
					mu               sync.Mutex
					appliedResources = map[string][]byte{}
					appliedEvents    = make(chan dogma.Event, 100)
				)

				deps.Handler.HandleEventFunc = func(
					ctx context.Context,
					r, c, n []byte,
					s dogma.ProjectionEventScope,
					e dogma.Event,
				) (bool, error) {
					mu.Lock()
					defer mu.Unlock()

					v := appliedResources[string(r)]
					if !bytes.Equal(v, c) {
						t.Logf("[%T] resource %x occ conflict: %x != %x", e, r, c, v)
						return false, nil
					}

					select {
					case <-ctx.Done():
						return false, ctx.Err()
					case appliedEvents <- e:
						t.Logf("[%T] resource %x updated: %x -> %x", e, r, c, n)
						appliedResources[string(r)] = n
						return true, nil
					}
				}

				deps.Handler.ResourceVersionFunc = func(ctx context.Context, r []byte) ([]byte, error) {
					mu.Lock()
					defer mu.Unlock()

					v := appliedResources[string(r)]
					t.Logf("resource %x loaded: %x", r, v)

					return v, nil
				}

				expectedStreamID := uuidpb.Generate()
				expectedEvents := []*envelopepb.Envelope{
					deps.Packer.Pack(MessageE1),
					deps.Packer.Pack(MessageE2),
					deps.Packer.Pack(MessageE3),
				}

				deps.EventConsumer.ConsumeFunc = func(
					ctx context.Context,
					streamID *uuidpb.UUID,
					offset eventstream.Offset,
					events chan<- eventstream.Event,
				) error {
					var matching []*envelopepb.Envelope

					if streamID.Equal(expectedStreamID) && offset < eventstream.Offset(len(expectedEvents)) {
						matching = expectedEvents[offset:]
					}

					for i, env := range matching {
						ese := eventstream.Event{
							StreamID: streamID,
							Offset:   eventstream.Offset(i),
							Envelope: env,
						}

						select {
						case <-ctx.Done():
							return ctx.Err()
						case events <- ese:
						}
					}

					<-ctx.Done()
					return ctx.Err()
				}

				deps.Supervisor.StreamIDs = []*uuidpb.UUID{expectedStreamID}

				c.InduceFailure(&deps)

				supervisorTask := test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					RepeatedlyUntilSuccess()

				for _, env := range expectedEvents {
					expected, err := deps.Packer.Unpack(env)
					if err != nil {
						t.Fatal(err)
					}

					test.
						ExpectChannelToReceive(
							tctx,
							appliedEvents,
							expected,
						)
				}

				test.
					ExpectChannelWouldBlock(
						tctx,
						appliedEvents,
					)
				deps.Supervisor.Shutdown()
				supervisorTask.WaitForSuccess()
			})
		}
	})

	t.Run("it makes the event type available via the scope", func(t *testing.T) {
		tctx := test.WithContext(t)

		deps := setup(tctx)

		env := deps.Packer.Pack(MessageE1)

		var supervisorTask *test.Task

		deps.Handler.HandleEventFunc = func(
			ctx context.Context,
			r, c, n []byte,
			s dogma.ProjectionEventScope,
			e dogma.Event,
		) (bool, error) {
			expected := env.GetCreatedAt().AsTime()
			if !s.RecordedAt().Equal(expected) {
				t.Fatalf("unexpected recorded at time: got %s, want %s", s.RecordedAt(), expected)
			}

			supervisorTask.Stop()

			return true, nil
		}

		deps.EventConsumer.ConsumeFunc = func(
			ctx context.Context,
			streamID *uuidpb.UUID,
			offset eventstream.Offset,
			events chan<- eventstream.Event,
		) error {
			ese := eventstream.Event{
				StreamID: streamID,
				Offset:   0,
				Envelope: env,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case events <- ese:
			}

			<-ctx.Done()
			return ctx.Err()
		}

		deps.Supervisor.StreamIDs = []*uuidpb.UUID{uuidpb.Generate()}

		supervisorTask = test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilStopped()
		supervisorTask.WaitUntilStopped()
	})
}
