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
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
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

		var actual atomic.Value
		deps.eventRecorder.RecordEventsFunc = func(envs []*envelopepb.Envelope) {
			actual.Store(envs)
			deps.cancel()
		}

		result := make(chan error, 1)
		go func() {
			result <- deps.supervisor.Run(deps.ctx)
		}()

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		err := <-result
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}

		if diff := cmp.Diff(
			actual.Load(),
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
			protocmp.Transform(),
		); diff != "" {

			t.Fatal(diff)
		}
	})
	t.Run("it does not re-handle successful commands after restart", func(t *testing.T) {
		deps := setup(t)

		var handledCount atomic.Uint64
		deps.handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			handledCount.Add(1)
			return nil
		}

		result := make(chan error, 1)
		go func() {
			result <- deps.supervisor.Run(deps.ctx)
		}()

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		deps.cancel()

		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}

		secondRunCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*250)
		defer cancel()

		go func() {
			result <- deps.supervisor.Run(secondRunCtx)
		}()

		if err := <-result; !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}

		const expectedCalls = 1
		if handledCount.Load() != expectedCalls {
			t.Fatalf("unexpected number of calls to handler: got %d, want %d", handledCount.Load(), expectedCalls)
		}
	})

	t.Run("it recovers from a handler error", func(t *testing.T) {
		deps := setup(t)

		expectedErr := errors.New("<error>")
		handled := make(chan struct{})

		deps.handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			return expectedErr
		}

		result := make(chan error, 1)
		go func() {
			result <- deps.supervisor.Run(deps.ctx)
		}()

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		if err := <-result; !errors.Is(err, expectedErr) {
			t.Fatal(err)
		}

		deps.handler.HandleCommandFunc = func(ctx context.Context, ics dogma.IntegrationCommandScope, c dogma.Command) error {
			close(handled)
			return nil
		}
		go func() {
			result <- deps.supervisor.Run(deps.ctx)
		}()

		select {
		case err := <-result:
			t.Fatal(err)
		case <-handled:
			deps.cancel()
		}

		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
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

		result := make(chan error, 1)
		go func() {
			result <- deps.supervisor.Run(deps.ctx)
		}()

		if err := deps.executor.ExecuteCommand(deps.ctx, MessageC1); err != nil {
			t.Fatal(err)
		}

		select {
		case err := <-result:
			t.Fatal(err)
		case <-handled:
			deps.cancel()
		}

		err := <-result
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}
