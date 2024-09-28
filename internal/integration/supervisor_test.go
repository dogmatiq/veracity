package integration_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/enginekit/enginetest/stubs"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationjournal"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationkv"
	"github.com/dogmatiq/veracity/internal/test"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSupervisor(t *testing.T) {
	type dependencies struct {
		Packer        *envelope.Packer
		Journals      *memoryjournal.BinaryStore
		Keyspaces     *memorykv.BinaryStore
		Handler       *IntegrationMessageHandlerStub
		EventRecorder *eventRecorderStub
		Supervisor    *Supervisor
		Executor      *CommandExecutor
	}

	setup := func(test.TestingT) (deps dependencies) {
		deps.Packer = newPacker()

		deps.Journals = &memoryjournal.BinaryStore{}

		deps.Keyspaces = &memorykv.BinaryStore{}

		deps.Handler = &IntegrationMessageHandlerStub{}

		deps.EventRecorder = &eventRecorderStub{}

		deps.Supervisor = &Supervisor{
			HandlerIdentity: identitypb.New("<handler>", uuidpb.Generate()),
			Journals:        deps.Journals,
			Keyspaces:       deps.Keyspaces,
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
					test.FailOnJournalOpen(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before appending CommandEnqueued record to the journal",
				InduceFailure: func(deps *dependencies) {
					test.FailBeforeJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
							return r.GetCommandEnqueued() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending CommandEnqueued record to the journal",
				InduceFailure: func(deps *dependencies) {
					test.FailAfterJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
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
					test.FailBeforeJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
							return r.GetCommandHandled() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending CommandHandled record to the journal",
				InduceFailure: func(deps *dependencies) {
					test.FailAfterJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
							return r.GetCommandHandled() != nil
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
					) (eventstream.AppendResponse, error) {
						if done.CompareAndSwap(false, true) {
							return eventstream.AppendResponse{}, errors.New("<error>")
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
					) (eventstream.AppendResponse, error) {
						res, err := record(ctx, req)
						if err != nil {
							return eventstream.AppendResponse{}, err
						}

						if done.CompareAndSwap(false, true) {
							return eventstream.AppendResponse{}, errors.New("<error>")
						}

						return res, nil
					}
				},
			},
			{
				Desc:                              "failure before appending EventsAppendedToStream record to the journal",
				ExpectMultipleEventAppendRequests: true,
				InduceFailure: func(deps *dependencies) {
					test.FailBeforeJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
							return r.GetEventsAppendedToStream() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending EventsAppendedToStream record to the journal",
				InduceFailure: func(deps *dependencies) {
					test.FailAfterJournalAppend(
						deps.Journals,
						integrationjournal.Name(deps.Supervisor.HandlerIdentity.Key),
						func(r *integrationjournal.Record) bool {
							return r.GetEventsAppendedToStream() != nil
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before adding command to handled-commands keyspace",
				InduceFailure: func(deps *dependencies) {
					test.FailBeforeKeyspaceSet(
						deps.Keyspaces,
						integrationkv.HandledCommandsKeyspace(deps.Supervisor.HandlerIdentity.Key),
						func(k, v []byte) bool {
							return true
						},
					)
				},
			},
			{
				Desc: "failure after adding command to handled-commands keyspace",
				InduceFailure: func(deps *dependencies) {
					test.FailAfterKeyspaceSet(
						deps.Keyspaces,
						integrationkv.HandledCommandsKeyspace(deps.Supervisor.HandlerIdentity.Key),
						func(k, v []byte) bool {
							return true
						},
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
					if c != CommandA1 {
						return fmt.Errorf("unexpected command: got %s, want %s", c, CommandA1)
					}

					s.RecordEvent(EventA1)
					s.RecordEvent(EventA2)
					s.RecordEvent(EventA3)

					handled <- struct{}{}

					return nil
				}

				appendRequests := make(chan eventstream.AppendRequest, 100)
				deps.EventRecorder.AppendEventsFunc = func(
					ctx context.Context,
					req eventstream.AppendRequest,
				) (eventstream.AppendResponse, error) {
					appendRequests <- req
					return eventstream.AppendResponse{}, nil
				}

				var isFirstSelect atomic.Bool
				streamID := uuidpb.Generate()
				lowestPossibleOffset := eventstream.Offset(1)
				deps.EventRecorder.SelectEventStreamFunc = func(
					ctx context.Context,
				) (*uuidpb.UUID, eventstream.Offset, error) {
					if isFirstSelect.CompareAndSwap(false, true) {
						return streamID, lowestPossibleOffset, nil
					}
					return uuidpb.Generate(), eventstream.Offset(rand.Uint32()), nil
				}

				cmd := deps.Packer.Pack(CommandA1)

				c.InduceFailure(&deps)

				supervisor := test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					RepeatedlyUntilSuccess()

				req := ExecuteRequest{
					Command: cmd,
				}

				for {
					_, err := deps.Supervisor.ExecuteQueue.Do(tctx, req)

					if tctx.Err() != nil {
						t.Fatal(tctx.Err())
					}

					if err == nil {
						break
					}

				}

				expectedAppendRequest := eventstream.AppendRequest{
					StreamID: streamID,
					Events: []*envelopepb.Envelope{
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       "event(stubs.TypeA:A1, valid)",
							PortableName:      "EventStub[TypeA]",
							MediaType:         `application/json; type="EventStub[TypeA]"`,
							Data:              []byte(`{"content":"A1"}`),
						},
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       "event(stubs.TypeA:A2, valid)",
							PortableName:      "EventStub[TypeA]",
							MediaType:         `application/json; type="EventStub[TypeA]"`,
							Data:              []byte(`{"content":"A2"}`),
						},
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       "event(stubs.TypeA:A3, valid)",
							PortableName:      "EventStub[TypeA]",
							MediaType:         `application/json; type="EventStub[TypeA]"`,
							Data:              []byte(`{"content":"A3"}`),
						},
					},
					LowestPossibleOffset: lowestPossibleOffset,
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

				appendRequest := test.
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
					)

				deps.Supervisor.Shutdown()
				supervisor.WaitForSuccess()
				close(appendRequests)

				if c.ExpectMultipleEventAppendRequests {
					test.ExpectChannelToReceive(
						t,
						appendRequests,
						appendRequest,
					)

					for req := range appendRequests {
						test.Expect(
							tctx,
							"unexpected event stream append request",
							req,
							appendRequest,
						)
					}
				} else {
					test.ExpectChannelToClose(
						tctx,
						appendRequests,
					)
				}

				// TODO: look into resetting the latch
				secondSupervisor := &Supervisor{
					HandlerIdentity: deps.Supervisor.HandlerIdentity,
					Journals:        deps.Journals,
					Keyspaces:       deps.Keyspaces,
					Packer:          deps.Packer,
					Handler:         deps.Handler,
					EventRecorder:   deps.EventRecorder,
				}

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
					RunInBackground(t, "supervisor", secondSupervisor.Run).
					BeforeTestEnds()

				if _, err := secondSupervisor.ExecuteQueue.Do(tctx, req); err != nil {
					t.Fatal(err)
				}

				secondSupervisor.Shutdown()

				supervisor.WaitForSuccess()

				test.
					ExpectChannelWouldBlock(
						tctx,
						rehandled,
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
			BeforeTestEnds()

		deps.Handler.HandleCommandFunc = func(
			context.Context,
			dogma.IntegrationCommandScope,
			dogma.Command,
		) error {
			deps.Supervisor.Shutdown()
			return nil
		}

		if err := deps.Executor.ExecuteCommand(tctx, CommandA1); err != nil {
			t.Fatal(err)
		}

		supervisor.WaitForSuccess()

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

	t.Run("it increases the lowest possible offset after appending events", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		appendRequests := make(chan eventstream.AppendRequest, 100)
		var endOffset atomic.Uint64

		endOffset.Store(10)

		deps.EventRecorder.AppendEventsFunc = func(
			ctx context.Context,
			req eventstream.AppendRequest,
		) (eventstream.AppendResponse, error) {
			appendRequests <- req
			numEvents := uint64(len(req.Events))
			endOffset := endOffset.Add(numEvents)
			return eventstream.AppendResponse{
				EndOffset:   eventstream.Offset(endOffset),
				BeginOffset: eventstream.Offset(endOffset - numEvents),
			}, nil
		}

		streamID := uuidpb.Generate()
		deps.EventRecorder.SelectEventStreamFunc = func(
			ctx context.Context,
		) (*uuidpb.UUID, eventstream.Offset, error) {
			return streamID, eventstream.Offset(endOffset.Load()), nil
		}

		test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilTestEnds()

		deps.Handler.HandleCommandFunc = func(
			_ context.Context,
			s dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			switch c {
			case CommandA1:
				s.RecordEvent(EventA1)
			case CommandA2:
				s.RecordEvent(EventA2)
			}
			return nil
		}

		if err := deps.Executor.ExecuteCommand(tctx, CommandA1); err != nil {
			t.Fatal(err)
		}
		if err := deps.Executor.ExecuteCommand(tctx, CommandA2); err != nil {
			t.Fatal(err)
		}

		test.
			ExpectChannelToReceive(
				t,
				appendRequests,
				eventstream.AppendRequest{
					StreamID: streamID,
					Events: []*envelopepb.Envelope{
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       `event(stubs.TypeA:A1, valid)`,
							PortableName:      `EventStub[TypeA]`,
							MediaType:         `application/json; type="EventStub[TypeA]"`,
							Data:              []byte(`{"content":"A1"}`),
						},
					},
					LowestPossibleOffset: 10,
				},
				protocmp.IgnoreFields(
					&envelopepb.Envelope{},
					"message_id",
					"causation_id",
					"correlation_id",
				),
			)

		test.
			ExpectChannelToReceive(
				t,
				appendRequests,
				eventstream.AppendRequest{
					StreamID: streamID,
					Events: []*envelopepb.Envelope{
						{
							SourceApplication: deps.Packer.Application,
							SourceHandler:     deps.Supervisor.HandlerIdentity,
							CreatedAt:         timestamppb.New(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
							Description:       `event(stubs.TypeA:A2, valid)`,
							PortableName:      `EventStub[TypeA]`,
							MediaType:         `application/json; type="EventStub[TypeA]"`,
							Data:              []byte(`{"content":"A2"}`),
						},
					},
					LowestPossibleOffset: 11,
				},
				protocmp.IgnoreFields(
					&envelopepb.Envelope{},
					"message_id",
					"causation_id",
					"correlation_id",
				),
			)
	})
}
