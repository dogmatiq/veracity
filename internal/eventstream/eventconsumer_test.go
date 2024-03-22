package eventstream_test

import (
	"errors"
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

		deps.Consumer = &EventConsumer{}

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

		res, err := deps.Supervisor.AppendQueue.Exchange(
			tctx,
			AppendRequest{
				StreamID: streamID,
				Events: []*envelopepb.Envelope{
					deps.Packer.Pack(MessageE1),
					deps.Packer.Pack(MessageE2),
					deps.Packer.Pack(MessageE3),
				},
				IsFirstAttempt: true,
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		err = deps.Consumer.Consume(tctx, streamID, 0)
		if err != nil {
			t.Fatal(err)
		}

		test.Expect(
			t,
			"unexpected offset",
			offset,
			0,
		)

		ev := deps.Packer.Pack(MessageE1)

		res, err := deps.Recorder.AppendEvents(
			tctx,
			AppendRequest{
				StreamID: streamID,
				Events: []*envelopepb.Envelope{
					ev,
				},
				IsFirstAttempt: true,
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		test.Expect(
			t,
			"unexpected response",
			res,
			AppendResponse{
				BeginOffset: 0,
				EndOffset:   1,
			},
		)

		test.ExpectChannelToReceive(
			t,
			deps.Events,
			Event{
				StreamID: streamID,
				Offset:   0,
				Envelope: ev,
			},
		)
		deps.Supervisor.Shutdown()
		supervisor.StopAndWait()

	})

	t.Run("it propagates failures", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		streamID, offset, err := deps.Recorder.SelectEventStream(tctx)
		if err != nil {
			t.Fatal(err)
		}

		test.Expect(
			t,
			"unexpected offset",
			offset,
			0,
		)

		ev := deps.Packer.Pack(MessageE1)

		test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			FailBeforeTestEnds()

		test.FailOnJournalOpen(deps.Journals, JournalName(streamID), errors.New("<error>"))

		_, err = deps.Recorder.AppendEvents(
			tctx,
			AppendRequest{
				StreamID: streamID,
				Events: []*envelopepb.Envelope{
					ev,
				},
				IsFirstAttempt: true,
			},
		)
		if err == nil {
			t.Fatal("expected err")
		}
	})
}
