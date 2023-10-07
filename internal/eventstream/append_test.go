package eventstream_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/protobuf/protojournal"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/dogmatiq/veracity/persistence/journal"
)

func TestAppend(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memory.JournalStore
		Supervisor *Supervisor
		Events     <-chan Event
		Packer     *envelope.Packer
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memory.JournalStore{}

		events := make(chan Event, 100)

		deps.Supervisor = &Supervisor{
			Journals: deps.Journals,
			Events:   events,
			Logger:   test.NewLogger(t),
		}

		deps.Events = events

		deps.Packer = &envelope.Packer{
			Application: identitypb.New("<app>", uuidpb.Generate()),
			Marshaler:   Marshaler,
		}

		return deps
	}

	t.Run("it does not duplicate events", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()

		cases := []struct {
			Desc          string
			InduceFailure func(*dependencies)
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
						JournalName(streamID),
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before appending to journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailBeforeJournalAppend(
						deps.Journals,
						JournalName(streamID),
						func(*journalpb.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending to journal",
				InduceFailure: func(deps *dependencies) {
					memory.FailAfterJournalAppend(
						deps.Journals,
						JournalName(streamID),
						func(*journalpb.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
				},
			},
		}

		for _, c := range cases {
			c := c // capture loop variable

			t.Run(c.Desc, func(t *testing.T) {
				tctx := test.WithContext(t)
				deps := setup(tctx)

				t.Log("append some initial events to the stream")

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

				supervisor.StopAndWait()

				t.Log("induce a failure")

				c.InduceFailure(&deps)

				supervisor = test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					RepeatedlyUntilStopped()

				event := deps.Packer.Pack(MessageE1)

				req := AppendRequest{
					StreamID:       streamID,
					Events:         []*envelopepb.Envelope{event},
					IsFirstAttempt: true,
				}

				attempt := 1
				for {
					t.Logf("append an event, attempt #%d", attempt)
					attempt++

					_, err := deps.Supervisor.AppendQueue.Exchange(tctx, req)
					if err == nil {
						break
					}

					req.IsFirstAttempt = false
					req.LowestPossibleOffset = res.EndOffset
				}

				t.Log("stop the supervisor gracefully")

				deps.Supervisor.Shutdown()
				supervisor.StopAndWait()

				t.Log("ensure that the event was appended to the stream exactly once")

				j, err := deps.Journals.Open(tctx, JournalName(streamID))
				if err != nil {
					t.Fatal(err)
				}

				var events []*envelopepb.Envelope

				if err := protojournal.Range(
					tctx,
					j,
					1,
					func(
						ctx context.Context,
						_ journal.Position,
						r *journalpb.Record,
					) (bool, error) {
						events = append(events, r.GetAppendOperation().GetEvents()...)
						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"journal contains missing or unexpected events",
					events,
					[]*envelopepb.Envelope{
						event,
					},
				)
			})
		}
	})
}
