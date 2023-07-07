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
	setup := func(t *testing.T) (result struct {
		ctx           context.Context
		cancel        context.CancelFunc
		packer        *envelope.Packer
		supervisor    *Supervisor
		handler       *IntegrationMessageHandler
		exchanges     chan *EnqueueCommandExchange
		eventRecorder *eventRecorderStub
		executor      *CommandExecutor
	}) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		t.Cleanup(cancel)
		result.ctx = ctx
		result.cancel = cancel
		result.packer = newPacker()
		result.exchanges = make(chan *EnqueueCommandExchange)
		result.handler = &IntegrationMessageHandler{}
		result.eventRecorder = &eventRecorderStub{}
		result.executor = &CommandExecutor{
			EnqueueCommands: result.exchanges,
			Packer:          result.packer,
		}

		result.supervisor = &Supervisor{
			EnqueueCommand:  result.exchanges,
			HandlerIdentity: identitypb.New("<handler>", uuidpb.Generate()),
			Journals:        &memory.JournalStore{},
			Packer:          result.packer,
			Handler:         result.handler,
			EventRecorder:   result.eventRecorder,
		}

		return result
	}
	t.Run("it records events to the event stream", func(t *testing.T) {
		deps := setup(t)

		deps.handler.HandleCommandFunc = func(
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
		deps.eventRecorder.RecordEventsFunc = func(envs []*envelopepb.Envelope) {
			events <- envs
		}

		test.RunUntilTestEnds(t, deps.supervisor.Run)

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		test.ExpectToReceive(
			deps.ctx,
			t,
			events,
			[]*envelopepb.Envelope{
				{
					MessageId:         deterministicUUID(2),
					CausationId:       deterministicUUID(1),
					CorrelationId:     deterministicUUID(1),
					SourceApplication: deps.packer.Application,
					SourceHandler:     deps.supervisor.HandlerIdentity,
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
					SourceApplication: deps.packer.Application,
					SourceHandler:     deps.supervisor.HandlerIdentity,
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
					SourceApplication: deps.packer.Application,
					SourceHandler:     deps.supervisor.HandlerIdentity,
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
		deps := setup(t)

		result, cancel := test.Run(t, deps.supervisor.Run)

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		cancel()

		test.ExpectToReceive(deps.ctx, t, result, context.Canceled)

		handled := make(chan struct{})
		deps.handler.HandleCommandFunc = func(ctx context.Context, ics dogma.IntegrationCommandScope, c dogma.Command) error {
			close(handled)
			return nil
		}

		test.RunUntilTestEnds(t, deps.supervisor.Run)

		select {
		case <-handled:
			t.Fatal("handled command that was already handled successfully")
		case <-deps.ctx.Done():
			// all good
		}
	})

	t.Run("it recovers from a handler error", func(t *testing.T) {
		deps := setup(t)

		expectedErr := errors.New("<error>")

		deps.handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			return expectedErr
		}

		result, _ := test.Run(t, deps.supervisor.Run)

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		test.ExpectToReceive(deps.ctx, t, result, expectedErr)

		handled := make(chan struct{})
		deps.handler.HandleCommandFunc = func(ctx context.Context, ics dogma.IntegrationCommandScope, c dogma.Command) error {
			close(handled)
			return nil
		}

		test.RunUntilTestEnds(t, deps.supervisor.Run)

		test.ExpectToClose(deps.ctx, t, handled)
	})

	t.Run("it passes the command to the handler", func(t *testing.T) {
		deps := setup(t)

		handled := make(chan struct{})

		deps.handler.HandleCommandFunc = func(
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

		test.RunUntilTestEnds(t, deps.supervisor.Run)

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		test.ExpectToClose(deps.ctx, t, handled)
	})
}
