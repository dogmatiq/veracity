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
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
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
	packer := &envelope.Packer{
		Application: identitypb.New("<app>", uuidpb.Generate()),
		Marshaler:   Marshaler,
	}

	newSupervisor := func() (*integration.Supervisor, *IntegrationMessageHandler, chan *integration.EnqueueCommandExchange) {
		exchanges := make(chan *integration.EnqueueCommandExchange)
		handler := &IntegrationMessageHandler{}

		s := &integration.Supervisor{
			EnqueueCommand: exchanges,
			HandlerKey:     "faf5be87-e753-407c-a044-e72e4c5bf082",
			Journals:       &memory.JournalStore{},
			Packer:         packer,
			Handler:        handler,
		}
		return s, handler, exchanges
	}
	t.Run("it does not re-handle successful commands after restart", func(t *testing.T) {
		env := packer.Pack(MessageC1)

		enqueued := make(chan struct{})
		ex := &integration.EnqueueCommandExchange{
			Command: env,
			Done:    enqueued,
		}
		var handledCount atomic.Uint64
		s, handler, exchanges := newSupervisor()
		handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			handledCount.Add(1)
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result := make(chan error, 1)
		go func() {
			result <- s.Run(ctx)
		}()

		select {
		case err := <-result:
			t.Fatal(err)
		case exchanges <- ex:
		}

		select {
		case err := <-result:
			t.Fatal(err)
		case <-enqueued:
			cancel()
		}

		err := <-result
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*250)
		defer cancel()

		go func() {
			result <- s.Run(ctx)
		}()

		err = <-result
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}

		const expectedCalls = 1
		if handledCount.Load() != expectedCalls {
			t.Fatalf("unexpected number of calls to handler: got %d, want %d", handledCount.Load(), expectedCalls)
		}
	})

	t.Run("it recovers from a handler error", func(t *testing.T) {
		expectedErr := errors.New("<error>")
		env := packer.Pack(MessageC1)

		enqueued := make(chan struct{})
		ex := &integration.EnqueueCommandExchange{
			Command: env,
			Done:    enqueued,
		}
		handled := make(chan struct{})

		s, handler, exchanges := newSupervisor()
		handler.HandleCommandFunc = func(
			_ context.Context,
			_ dogma.IntegrationCommandScope,
			c dogma.Command,
		) error {
			return expectedErr
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result := make(chan error, 1)
		go func() {
			result <- s.Run(ctx)
		}()

		select {
		case err := <-result:
			t.Fatal(err)
		case exchanges <- ex:
		}

		//TODO: fix race condition
		select {
		case err := <-result:
			t.Fatal(err)
		case <-enqueued:
		}

		err := <-result
		if !errors.Is(err, expectedErr) {
			t.Fatal(err)
		}

		handler.HandleCommandFunc = func(ctx context.Context, ics dogma.IntegrationCommandScope, c dogma.Command) error {
			close(handled)
			return nil
		}
		go func() {
			result <- s.Run(ctx)
		}()

		select {
		case err := <-result:
			t.Fatal(err)
		case <-handled:
			cancel()
		}
		err = <-result
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})

	t.Run("it passes the command to the handler", func(t *testing.T) {
		env := packer.Pack(MessageC1)

		enqueued := make(chan struct{})
		ex := &integration.EnqueueCommandExchange{
			Command: env,
			Done:    enqueued,
		}

		handled := make(chan struct{})

		s, handler, exchanges := newSupervisor()
		handler.HandleCommandFunc = func(
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

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result := make(chan error, 1)
		go func() {
			result <- s.Run(ctx)
		}()

		select {
		case err := <-result:
			t.Fatal(err)
		case exchanges <- ex:
		}

		select {
		case err := <-result:
			t.Fatal(err)
		case <-enqueued:
		}

		select {
		case err := <-result:
			t.Fatal(err)
		case <-handled:
			cancel()
		}

		err := <-result
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}
