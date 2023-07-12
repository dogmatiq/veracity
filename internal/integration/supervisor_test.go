package integration_test

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// - [x] Record command in inbox
// - [x] Invoke the handler associated with the command
// - [ ] Place all resulting events in the outbox and remove the command from the inbox
// - [ ] Dispatch event 1 in the outbox to the event stream
// - [ ] Contact event stream
// - [ ] Remove event 1 from the outbox
// - [ ] Dispatch event 2 in the outbox to the event stream
// - [ ] Remove event 2 from the outbox
// - [ ] Dispatch event 3 in the outbox to the event stream
// - [ ] Remove event 3 from the outbox
// - [ ] Remove integration from a dirty list.

func TestSupervisor(t *testing.T) {
	setup := func(t test.TestingT) (deps struct {
		Packer        *envelope.Packer
		Supervisor    *Supervisor
		Handler       *IntegrationMessageHandler
		Exchanges     chan *EnqueueCommandExchange
		EventRecorder *eventRecorderStub
		Executor      *CommandExecutor
	}) {
		deps.Packer = newPacker()

		deps.Exchanges = make(chan *EnqueueCommandExchange)

		deps.Handler = &IntegrationMessageHandler{}

		deps.EventRecorder = &eventRecorderStub{}

		deps.Executor = &CommandExecutor{
			EnqueueCommands: deps.Exchanges,
			Packer:          deps.Packer,
		}

		deps.Supervisor = &Supervisor{
			EnqueueCommand:  deps.Exchanges,
			HandlerIdentity: identitypb.New("<handler>", uuidpb.Generate()),
			Journals:        &memory.JournalStore{},
			Packer:          deps.Packer,
			Handler:         deps.Handler,
			EventRecorder:   deps.EventRecorder,
		}

		return deps
	}
	t.Run("it records events to the event stream", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		deps.Handler.HandleCommandFunc = func(
			_ context.Context,
			s dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			s.RecordEvent(MessageE1)
			s.RecordEvent(MessageE2)
			s.RecordEvent(MessageE3)

			return nil
		}

		events := make(chan []*envelopepb.Envelope, 1)
		deps.EventRecorder.RecordEventsFunc = func(envs []*envelopepb.Envelope) {
			events <- envs
		}

		test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilTestEnds()

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
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
	})

	t.Run("it does not re-handle successful commands after restart", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		supervisor := test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilStopped()

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		supervisor.StopAndWait()

		handled := make(chan struct{})
		deps.Handler.HandleCommandFunc = func(
			context.Context,
			dogma.IntegrationCommandScope,
			dogma.Command,
		) error {
			close(handled)
			return nil
		}

		test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilTestEnds()

		test.
			ExpectChannelToBlock(
				tctx,
				10*time.Second,
				handled,
			)
	})

	t.Run("it recovers from a handler error", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		handlerErr := errors.New("<error>")
		deps.Handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			return handlerErr
		}

		supervisor := test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilStopped()

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		test.
			ExpectChannelToClose(
				tctx,
				supervisor.Done(),
			)

		if err := supervisor.Err(); err != handlerErr {
			t.Fatalf("unexpected error: %s", err)
		}

		handled := make(chan struct{})
		deps.Handler.HandleCommandFunc = func(ctx context.Context, ics dogma.IntegrationCommandScope, c dogma.Command) error {
			close(handled)
			return nil
		}

		test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilTestEnds()

		test.
			ExpectChannelToClose(
				tctx,
				handled,
			)
	})

	t.Run("it passes the command to the handler", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		handled := make(chan struct{})
		deps.Handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			close(handled)

			if c != MessageC1 {
				return fmt.Errorf("unexpected command: got %s, want %s", c, MessageC1)
			}

			return nil
		}

		test.
			RunInBackground(t, deps.Supervisor.Run).
			UntilTestEnds()

		if err := deps.Executor.ExecuteCommand(tctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		test.
			ExpectChannelToClose(
				tctx,
				handled,
			)
	})
}
