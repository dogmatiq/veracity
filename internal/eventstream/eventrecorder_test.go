package eventstream_test

import (
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

func TestEventRecorder(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memoryjournal.Store
		Supervisor *Supervisor
		Events     <-chan Event
		Packer     *envelope.Packer
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memoryjournal.Store{}

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

		return deps
	}

	t.Run("it appends the event to the stream", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		supervisor := test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilStopped()

		streamID := uuidpb.Generate()

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

		deps.Supervisor.Shutdown()
		supervisor.StopAndWait()

		test.Expect(
			t,
			"xxxxxxx",
			len(deps.Supervisor.Events),
			3,
		)
	})

	t.Run("it propagates failures", func(t *testing.T) {
		t.Parallel()

		// TODO
	})
}
