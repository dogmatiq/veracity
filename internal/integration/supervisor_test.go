package integration_test

import (
	"context"
	"errors"
	"fmt"
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

// - Record command in inbox
// - Invoke the handler associated with the command
// - Place all resulting events in the outbox and remove the command from the inbox
// - Dispatch event 1 in the outbox to the event stream
// - Contact event stream
// - Remove event 1 from the outbox
// - Dispatch event 2 in the outbox to the event stream
// - Remove event 2 from the outbox
// - Dispatch event 3 in the outbox to the event stream
// - Remove event 3 from the outbox
// - Remove integration from a dirty list.

func TestSupervisor(t *testing.T) {
	t.Run("it passes the command to the handler", func(t *testing.T) {
		packer := &envelope.Packer{
			Application: identitypb.New("<app>", uuidpb.Generate()),
			Marshaler:   Marshaler,
		}

		env := packer.Pack(MessageC1)

		enqueued := make(chan struct{})
		ex := &integration.EnqueueCommandExchange{
			Command: env,
			Done:    enqueued,
		}

		exchanges := make(chan *integration.EnqueueCommandExchange)
		handled := make(chan struct{})

		s := &integration.Supervisor{
			EnqueueCommand: exchanges,
			Handler: &IntegrationMessageHandler{
				HandleCommandFunc: func(
					_ context.Context,
					_ dogma.IntegrationCommandScope,
					c dogma.Command,
				) error {
					close(handled)

					if c != MessageC1 {
						return fmt.Errorf("unexpected command: got %s, want %s", c, MessageC1)
					}

					return nil
				},
			},
			HandlerKey: "faf5be87-e753-407c-a044-e72e4c5bf082",
			Journals:   &memory.JournalStore{},
			Packer:     packer,
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
