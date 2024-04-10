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
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/spruce"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/test"
)

func TestAppend(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memoryjournal.BinaryStore
		Supervisor *Supervisor
		Packer     *envelope.Packer
		Barrier    chan struct{}
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memoryjournal.BinaryStore{}

		deps.Supervisor = &Supervisor{
			Journals: deps.Journals,
			Logger:   spruce.NewLogger(t),
		}

		deps.Packer = &envelope.Packer{
			Application: identitypb.New("<app>", uuidpb.Generate()),
			Marshaler:   Marshaler,
		}

		deps.Barrier = make(chan struct{})

		return deps
	}

	t.Run("it does not duplicate events", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()

		cases := []struct {
			Desc          string
			InduceFailure func(context.Context, *testing.T, *dependencies)
		}{
			{
				Desc: "no faults",
			},
			{
				Desc: "failure to open journal",
				InduceFailure: func(_ context.Context, t *testing.T, deps *dependencies) {
					test.FailOnJournalOpen(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						errors.New("<error>"),
					)
					t.Log("configured journal store to fail when opening the journal")
					close(deps.Barrier)
				},
			},
			{
				Desc: "failure before appending to journal",
				InduceFailure: func(_ context.Context, t *testing.T, deps *dependencies) {
					test.FailBeforeJournalAppend(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						func(*eventstreamjournal.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
					t.Log("configured journal store to fail before appending a record")
					close(deps.Barrier)
				},
			},
			{
				Desc: "failure after appending to journal",
				InduceFailure: func(_ context.Context, t *testing.T, deps *dependencies) {
					test.FailAfterJournalAppend(
						deps.Journals,
						eventstreamjournal.Name(streamID),
						func(*eventstreamjournal.Record) bool {
							return true
						},
						errors.New("<error>"),
					)
					t.Log("configured journal store to fail after appending a record")
					close(deps.Barrier)
				},
			},
			{
				Desc: "optimistic concurrency conflict",
				InduceFailure: func(ctx context.Context, t *testing.T, deps *dependencies) {
					go func() {
						if _, err := deps.Supervisor.AppendQueue.Do(
							ctx,
							AppendRequest{
								StreamID: streamID,
								Events: []*envelopepb.Envelope{
									deps.Packer.Pack(MessageX1),
								},
							},
						); err != nil {
							t.Error(err)
							return
						}

						t.Log("confirmed that the supervisor-under-test is running")

						s := &Supervisor{
							Journals: deps.Journals,
							Logger:   spruce.NewLogger(t),
						}

						defer test.
							RunInBackground(t, "conflict-generating-supervisor", s.Run).
							UntilStopped().
							Stop()

						if _, err := s.AppendQueue.Do(
							ctx,
							AppendRequest{
								StreamID: streamID,
								Events: []*envelopepb.Envelope{
									deps.Packer.Pack(MessageX2),
								},
							},
						); err != nil {
							t.Error(err)
							return
						}

						t.Log("appended events using a different supervisor to induce a journal conflict")

						close(deps.Barrier)
					}()
				},
			},
		}

		for _, c := range cases {
			t.Run(c.Desc, func(t *testing.T) {
				tctx := test.WithContext(t)
				deps := setup(tctx)

				t.Log("seeding the event stream with some initial events")

				supervisor := test.
					RunInBackground(t, "event-seeding-supervisor", deps.Supervisor.Run).
					UntilStopped()

				res, err := deps.Supervisor.AppendQueue.Do(
					tctx,
					AppendRequest{
						StreamID: streamID,
						Events: []*envelopepb.Envelope{
							deps.Packer.Pack(MessageE1),
							deps.Packer.Pack(MessageE2),
							deps.Packer.Pack(MessageE3),
						},
					},
				)
				if err != nil {
					t.Fatal(err)
				}

				supervisor.StopAndWait()

				// Open a journal that was can use for verifying results
				// _before_ inducing any failure.
				j, err := eventstreamjournal.Open(tctx, deps.Journals, streamID)
				if err != nil {
					t.Fatal(err)
				}
				defer j.Close()

				if c.InduceFailure != nil {
					c.InduceFailure(tctx, t, &deps)
				} else {
					close(deps.Barrier)
				}

				supervisor = test.
					RunInBackground(t, "supervisor-under-test", deps.Supervisor.Run).
					RepeatedlyUntilStopped()

				<-deps.Barrier

				// Read the journal bounds as they exist before the test
				// commences.
				begin, end, err := j.Bounds(tctx)
				if err != nil {
					t.Fatal(err)
				}

				t.Logf("proceeding with test, journal bounds are [%d, %d)", begin, end)

				event := deps.Packer.Pack(MessageE1)

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

				var events []*envelopepb.Envelope

				if err := j.Range(
					tctx,
					end, // only read the records appended during the test
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
