package eventstream_test

import (
	"context"
	"testing"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/spruce"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/test"
)

func TestEventConsumer(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memoryjournal.BinaryStore
		Supervisor *Supervisor
		Events     <-chan Event
		Packer     *envelope.Packer
		Consumer   *EventConsumer
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memoryjournal.BinaryStore{}

		events := make(chan Event, 100)

		deps.Supervisor = &Supervisor{
			Journals: deps.Journals,
			Events:   events,
			Logger:   spruce.NewLogger(t),
		}

		deps.Events = events

		deps.Packer = &envelope.Packer{
			Application: identitypb.New("<app>", uuidpb.Generate()),
			Marshaler:   Marshaler,
		}

		deps.Consumer = &EventConsumer{
			Events: events,
		}

		return deps
	}

	t.Run("it reads an event from the stream at 0 offset", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()
		tctx := test.WithContext(t)
		deps := setup(tctx)

		supervisor := test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilStopped()

		ev1 := deps.Packer.Pack(MessageE1)
		ev2 := deps.Packer.Pack(MessageE2)
		ev3 := deps.Packer.Pack(MessageE3)

		if _, err := deps.Supervisor.AppendQueue.Exchange(
			tctx,
			AppendRequest{
				StreamID:       streamID,
				Events:         []*envelopepb.Envelope{ev1, ev2, ev3},
				IsFirstAttempt: true,
			},
		); err != nil {
			t.Fatal(err)
		}

		events := make(chan Event, 100)
		consumer := test.
			RunInBackground(
				t,
				"consumer",
				func(ctx context.Context) error {
					return deps.Consumer.Consume(ctx, streamID, 0, events)
				},
			).
			UntilStopped()

		test.ExpectChannelToReceive(
			t,
			events,
			Event{
				StreamID: streamID,
				Offset:   0,
				Envelope: ev1,
			},
		)

		test.ExpectChannelToReceive(
			t,
			events,
			Event{
				StreamID: streamID,
				Offset:   1,
				Envelope: ev2,
			},
		)

		test.ExpectChannelToReceive(
			t,
			events,
			Event{
				StreamID: streamID,
				Offset:   2,
				Envelope: ev3,
			},
		)

		deps.Supervisor.Shutdown()
		supervisor.StopAndWait()
		consumer.StopAndWait()
	})
}
