package eventstream_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/dogmatiq/enginekit/enginetest/stubs"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/spruce"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/test"
)

func TestAppend(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memoryjournal.BinaryStore
		Supervisor *Supervisor
		Packer     *envelopepb.Packer
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memoryjournal.BinaryStore{}

		deps.Supervisor = &Supervisor{
			Journals: deps.Journals,
			Logger:   spruce.NewTestLogger(t),
		}

		deps.Packer = &envelopepb.Packer{
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
					test.FailOnJournalOpen(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure before appending to journal",
				InduceFailure: func(deps *dependencies) {
					test.FailBeforeJournalAppend(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						func(*eventstreamjournal.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
				},
			},
			{
				Desc: "failure after appending to journal",
				InduceFailure: func(deps *dependencies) {
					test.FailAfterJournalAppend(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						func(*eventstreamjournal.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
				},
			},
		}

		for _, c := range cases {
			t.Run(c.Desc, func(t *testing.T) {
				tctx := test.WithContext(t)
				deps := setup(tctx)

				t.Log("append some initial events to the stream")

				supervisor := test.
					RunInBackground(t, "supervisor", deps.Supervisor.Run).
					UntilStopped()

				res, err := deps.Supervisor.AppendQueue.Do(
					tctx,
					AppendRequest{
						StreamID: streamID,
						Events: []*envelopepb.Envelope{
							deps.Packer.Pack(EventE1),
							deps.Packer.Pack(EventE2),
							deps.Packer.Pack(EventE3),
						},
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

				event := deps.Packer.Pack(EventE1)

				req := AppendRequest{
					StreamID: streamID,
					Events:   []*envelopepb.Envelope{event},
				}

				attempt := 1
				for {
					t.Logf("append an event, attempt #%d", attempt)
					attempt++

					_, err := deps.Supervisor.AppendQueue.Do(tctx, req)
					if err == nil {
						break
					}

					req.LowestPossibleOffset = res.EndOffset
				}

				t.Log("stop the supervisor gracefully")

				deps.Supervisor.Shutdown()
				supervisor.StopAndWait()

				t.Log("ensure that the event was appended to the stream exactly once")

				j, err := eventstreamjournal.Open(tctx, deps.Journals, streamID)
				if err != nil {
					t.Fatal(err)
				}

				var events []*envelopepb.Envelope

				if err := j.Range(
					tctx,
					1,
					func(
						ctx context.Context,
						_ journal.Position,
						rec *eventstreamjournal.Record,
					) (bool, error) {
						events = append(
							events,
							rec.GetEventsAppended().GetEvents()...,
						)
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
