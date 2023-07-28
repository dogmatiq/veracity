package integration_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// - [x] Record command in inbox
// - [x] Invoke the handler associated with the command
// - [x] Place all resulting events in the outbox and remove the command from the inbox
// - [ ] Determine which event stream to use
// - [x] Record events to the event stream
// - [x] - Record events to the event stream after failure
// - [x] Remove events from the outbox
func TestSupervisor(t *testing.T) {
	type dependencies struct {
		Packer        *envelope.Packer
		Journals      *memory.JournalStore
		Supervisor    *Supervisor
		Handler       *IntegrationMessageHandler
		Exchanges     chan *EnqueueCommandExchange
		EventRecorder *eventRecorderStub
		Executor      *CommandExecutor
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Packer = newPacker()

		deps.Exchanges = make(chan *EnqueueCommandExchange)

		deps.Handler = &IntegrationMessageHandler{}

		deps.EventRecorder = &eventRecorderStub{}

		deps.Executor = &CommandExecutor{
			EnqueueCommands: deps.Exchanges,
			Packer:          deps.Packer,
		}

		deps.Journals = &memory.JournalStore{}

		deps.Supervisor = &Supervisor{
			EnqueueCommand:  deps.Exchanges,
			HandlerIdentity: identitypb.New("<handler>", uuidpb.Generate()),
			Journals:        deps.Journals,
			Packer:          deps.Packer,
			Handler:         deps.Handler,
			EventRecorder:   deps.EventRecorder,
		}

		return deps
	}

	t.Run("it executes commands once regardless of errors", func(t *testing.T) {
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
				Desc: "failure to open journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailOnJournalOpen(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before appending CommandEnqueued record to the journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailBeforeJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandEnqueued() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending CommandEnqueued record to the journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailAfterJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandEnqueued() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before handler records events",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleCommandFunc

					deps.Handler.HandleCommandFunc = func(
						ctx context.Context,
						s dogma.IntegrationCommandScope,
						c dogma.Command,
					) error {
						if done.CompareAndSwap(false, true) {
							return errors.New("<error>")
						}

						if err := handle(ctx, s, c); err != nil {
							return err
						}

						return nil
					}
				},
			},
			{
				Desc: "failure after handler records events",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					handle := deps.Handler.HandleCommandFunc

					deps.Handler.HandleCommandFunc = func(
						ctx context.Context,
						s dogma.IntegrationCommandScope,
						c dogma.Command,
					) error {
						if err := handle(ctx, s, c); err != nil {
							return err
						}

						if done.CompareAndSwap(false, true) {
							return errors.New("<error>")
						}
						return nil
					}
				},
			},
			{
				Desc: "failure before events recorded to stream",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					record := deps.EventRecorder.RecordEventsFunc

					deps.EventRecorder.RecordEventsFunc = func(
						events []*envelopepb.Envelope,
					) error {
						if done.CompareAndSwap(false, true) {
							return errors.New("<error>")
						}

						return record(events)

					}
				},
			},
			{
				Desc: "failure after events recorded to stream",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					record := deps.EventRecorder.RecordEventsFunc

					deps.EventRecorder.RecordEventsFunc = func(
						events []*envelopepb.Envelope,
					) error {
						if err := record(events); err != nil {
							return err
						}

						if done.CompareAndSwap(false, true) {
							return errors.New("<error>")
						}

						return nil
					}
				},
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.Desc, func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(tctx)

				handled := make(chan struct{}, 100)
				deps.Handler.HandleCommandFunc = func(
					_ context.Context,
					s dogma.IntegrationCommandScope,
					c dogma.Command,
				) error {
					if c != MessageC1 {
						return fmt.Errorf("unexpected command: got %s, want %s", c, MessageC1)
					}

					s.RecordEvent(MessageE1)
					s.RecordEvent(MessageE2)
					s.RecordEvent(MessageE3)

					handled <- struct{}{}

					return nil
				}

				events := make(chan []*envelopepb.Envelope, 100)
				deps.EventRecorder.RecordEventsFunc = func(envs []*envelopepb.Envelope) error {
					events <- envs
					return nil
				}
				cmd := deps.Packer.Pack(MessageC1)

				c.InduceFailure(&deps)

				test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					RepeatedlyUntilStopped()

			loop:
				for {
					done := make(chan error, 1)

					ex := &EnqueueCommandExchange{
						Command: cmd,
						Done:    done,
					}

					select {
					case <-tctx.Done():
						t.Fatal(tctx.Err())
					case deps.Exchanges <- ex:
					}

					select {
					case <-tctx.Done():
						t.Fatal(tctx.Err())
					case err := <-done:
						if err == nil {
							break loop
						}
					}
				}

				test.
					ExpectChannelToReceive(
						tctx,
						events,
						[]*envelopepb.Envelope{
							{
								MessageId:         deterministicUUID(2),
								CausationId:       deterministicUUID(1),
								CorrelationId:     deterministicUUID(1),
								SourceApplication: deps.Packer.Application,
								SourceHandler:     deps.Supervisor.HandlerIdentity,
								CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
								Description:       MessageE1.MessageDescription(),
								PortableName:      MessageEPortableName,
								MediaType:         MessageE1Packet.MediaType,
								Data:              MessageE1Packet.Data,
							},
							{
								MessageId:         deterministicUUID(3),
								CausationId:       deterministicUUID(1),
								CorrelationId:     deterministicUUID(1),
								SourceApplication: deps.Packer.Application,
								SourceHandler:     deps.Supervisor.HandlerIdentity,
								CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
								Description:       MessageE2.MessageDescription(),
								PortableName:      MessageEPortableName,
								MediaType:         MessageE2Packet.MediaType,
								Data:              MessageE2Packet.Data,
							},
							{
								MessageId:         deterministicUUID(4),
								CausationId:       deterministicUUID(1),
								CorrelationId:     deterministicUUID(1),
								SourceApplication: deps.Packer.Application,
								SourceHandler:     deps.Supervisor.HandlerIdentity,
								CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
								Description:       MessageE3.MessageDescription(),
								PortableName:      MessageEPortableName,
								MediaType:         MessageE3Packet.MediaType,
								Data:              MessageE3Packet.Data,
							},
						},
					)

				test.
					ExpectChannelToReceive(
						t,
						handled,
						struct{}{},
					)

				test.
					ExpectChannelWouldBlock(
						t,
						handled,
					)
			})
		}
	})

	t.Run("it does not re-handle successful commands after restart", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		supervisor := test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilStopped()

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		supervisor.StopAndWait()

		handled := make(chan struct{}, 100)
		deps.Handler.HandleCommandFunc = func(
			context.Context,
			dogma.IntegrationCommandScope,
			dogma.Command,
		) error {
			handled <- struct{}{}
			return nil
		}

		deps.EventRecorder.RecordEventsFunc = func(
			[]*envelopepb.Envelope,
		) error {
			handled <- struct{}{}
			return nil
		}

		test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilTestEnds()

		test.
			ExpectChannelToBlockForDuration(
				tctx,
				2*time.Second,
				handled,
			)
	})
}
