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
	"github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSupervisor(t *testing.T) {
	type dependencies struct {
		Packer        *envelope.Packer
		Journals      *memory.JournalStore
		Handler       *IntegrationMessageHandler
		EventRecorder *eventRecorderStub
		Supervisor    *Supervisor
		Executor      *CommandExecutor
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Packer = newPacker()

		deps.Journals = &memory.JournalStore{}

		deps.Handler = &IntegrationMessageHandler{}

		deps.EventRecorder = &eventRecorderStub{}

		deps.Supervisor = &Supervisor{
			HandlerIdentity: identitypb.New("<handler>", uuidpb.Generate()),
			Journals:        deps.Journals,
			Packer:          deps.Packer,
			Handler:         deps.Handler,
			EventRecorder:   deps.EventRecorder,
		}

		deps.Executor = &CommandExecutor{
			ExecuteQueue: &deps.Supervisor.ExecuteQueue,
			Packer:       deps.Packer,
		}

		return deps
	}

	t.Run("it executes commands once regardless of errors", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			Desc                              string
			InduceFailure                     func(*dependencies)
			ExpectMultipleEventAppendRequests bool
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

						return handle(ctx, s, c)
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
				Desc: "failure before appending CommandHandled record to the journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailBeforeJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandHandled() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending CommandHandled record to the journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailAfterJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandHandled() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before appending CommandHandlerFailed record to the journal",
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
						return handle(ctx, s, c)
					}

					memory.FailBeforeJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandHandlerFailed() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending CommandHandlerFailed record to the journal",
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
						return handle(ctx, s, c)
					}

					memory.FailAfterJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetCommandHandlerFailed() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before events appended to stream",
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					record := deps.EventRecorder.AppendEventsFunc

					deps.EventRecorder.AppendEventsFunc = func(
						ctx context.Context,
						req eventstream.AppendRequest,
					) error {
						if done.CompareAndSwap(false, true) {
							return errors.New("<error>")
						}

						return record(ctx, req)
					}
				},
			},
			{
				Desc:                              "failure after events appended to stream",
				ExpectMultipleEventAppendRequests: true,
				InduceFailure: func(deps *dependencies) {
					var done atomic.Bool
					record := deps.EventRecorder.AppendEventsFunc

					deps.EventRecorder.AppendEventsFunc = func(
						ctx context.Context,
						req eventstream.AppendRequest,
					) error {
						if err := record(ctx, req); err != nil {
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
				Desc:                              "failure before appending EventsAppendedToStream record to the journal",
				ExpectMultipleEventAppendRequests: true,
				InduceFailure: func(deps *dependencies) {
					memory.FailBeforeJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetEventsAppendedToStream() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending EventsAppendedToStream record to the journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailAfterJournalAppend(
						deps.Journals,
						JournalName(deps.Supervisor.HandlerIdentity.Key),
						func(r *journalpb.Record) bool {
							return r.GetEventsAppendedToStream() != nil
						},
						errors.New("<error>"),
					)
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

				appendRequests := make(chan eventstream.AppendRequest, 100)
				deps.EventRecorder.AppendEventsFunc = func(
					ctx context.Context,
					req eventstream.AppendRequest,
				) error {
					appendRequests <- req
					return nil
				}
				cmd := deps.Packer.Pack(MessageC1)

				c.InduceFailure(&deps)

				supervisor := test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					RepeatedlyUntilSuccess()

				for {
					_, err := deps.Supervisor.ExecuteQueue.Exchange(
						tctx,
						ExecuteRequest{Command: cmd},
					)

					if tctx.Err() != nil {
						t.Fatal(tctx.Err())
					}

					if err == nil {
						break
					}
				}

				expectedAppendRequest := eventstream.AppendRequest{
					Events: []*envelopepb.Envelope{
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       MessageE1.MessageDescription(),
							PortableName:      MessageEPortableName,
							MediaType:         MessageE1Packet.MediaType,
							Data:              MessageE1Packet.Data,
						},
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       MessageE2.MessageDescription(),
							PortableName:      MessageEPortableName,
							MediaType:         MessageE2Packet.MediaType,
							Data:              MessageE2Packet.Data,
						},
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       MessageE3.MessageDescription(),
							PortableName:      MessageEPortableName,
							MediaType:         MessageE3Packet.MediaType,
							Data:              MessageE3Packet.Data,
						},
					},
				}

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

				firstAppendRequest := test.
					ExpectChannelToReceive(
						t,
						appendRequests,
						expectedAppendRequest,
						protocmp.IgnoreFields(
							&envelopepb.Envelope{},
							"message_id",
							"causation_id",
							"correlation_id",
						),
						cmpopts.IgnoreFields(
							eventstream.AppendRequest{},
							"IsFirstAttempt",
						),
					)

				deps.Supervisor.Shutdown()
				supervisor.WaitForSuccess()
				close(appendRequests)

				if c.ExpectMultipleEventAppendRequests {
					subsequentAppendRequest := firstAppendRequest
					subsequentAppendRequest.IsFirstAttempt = false

					test.ExpectChannelToReceive(
						t,
						appendRequests,
						subsequentAppendRequest,
					)

					for req := range appendRequests {
						test.Expect(
							tctx,
							"unexpected event stream append request",
							req,
							subsequentAppendRequest,
						)
					}
				} else {
					test.ExpectChannelToClose(
						tctx,
						appendRequests,
					)
				}
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

		deps.Handler.HandleCommandFunc = func(
			context.Context,
			dogma.IntegrationCommandScope,
			dogma.Command,
		) error {
			supervisor.Stop()
			return nil
		}

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		supervisor.WaitUntilStopped()

		rehandled := make(chan struct{}, 100)
		deps.Handler.HandleCommandFunc = func(
			context.Context,
			dogma.IntegrationCommandScope,
			dogma.Command,
		) error {
			close(rehandled)
			return nil
		}

		supervisor = test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			BeforeTestEnds()

		deps.Supervisor.Shutdown()
		supervisor.WaitForSuccess()

		test.
			ExpectChannelWouldBlock(
				tctx,
				rehandled,
			)
	})
}
