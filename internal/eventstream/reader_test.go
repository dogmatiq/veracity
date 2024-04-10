package eventstream_test

import (
	"context"
	"testing"
	"time"

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

func TestReader(t *testing.T) {
	t.Parallel()

	type dependencies struct {
		Journals   *memoryjournal.BinaryStore
		Supervisor *Supervisor
		Packer     *envelope.Packer
		Reader     *Reader
	}

	setup := func(t test.TestingT) (deps dependencies) {
		deps.Journals = &memoryjournal.BinaryStore{}

		logger := spruce.NewLogger(t)

		deps.Supervisor = &Supervisor{
			Journals: deps.Journals,
			Logger:   logger,
		}

		deps.Packer = &envelope.Packer{
			Application: identitypb.New("<app>", uuidpb.Generate()),
			Marshaler:   Marshaler,
		}

		deps.Reader = &Reader{
			Journals:         deps.Journals,
			SubscribeQueue:   &deps.Supervisor.SubscribeQueue,
			UnsubscribeQueue: &deps.Supervisor.UnsubscribeQueue,
			Logger:           logger,
		}

		return deps
	}

	t.Run("it reads historical events from the journal", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()
		tctx := test.WithContext(t)
		deps := setup(tctx)

		supervisor := test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilStopped()

		env1 := deps.Packer.Pack(MessageE1)
		env2 := deps.Packer.Pack(MessageE2)
		env3 := deps.Packer.Pack(MessageE3)

		// Journal record with a single event.
		if _, err := deps.Supervisor.AppendQueue.Do(
			tctx,
			AppendRequest{
				StreamID: streamID,
				Events: []*envelopepb.Envelope{
					env1,
				},
			},
		); err != nil {
			t.Fatal(err)
		}

		// Journal record with multiple events.
		if _, err := deps.Supervisor.AppendQueue.Do(
			tctx,
			AppendRequest{
				StreamID: streamID,
				Events: []*envelopepb.Envelope{
					env2,
					env3,
				},
			},
		); err != nil {
			t.Fatal(err)
		}

		// Stop the supervisor now to verify that it's not necessary to read
		// historical events. We set the timeout to the minimum value possible
		// to reduce the test run-time.
		supervisor.StopAndWait()
		deps.Reader.SubscribeTimeout = 1

		events := make(chan Event)
		sub := &Subscriber{
			StreamID: streamID,
			Events:   events,
		}

		test.
			RunInBackground(t, "reader", func(ctx context.Context) error {
				return deps.Reader.Read(ctx, sub)
			}).
			UntilTestEnds()

		for offset, env := range []*envelopepb.Envelope{env1, env2, env3} {
			test.ExpectChannelToReceive(
				tctx,
				events,
				Event{
					StreamID: streamID,
					Offset:   Offset(offset),
					Envelope: env,
				},
			)
		}
	})

	t.Run("it reads contemporary events via a supervisor subscription", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()
		tctx := test.WithContext(t)
		deps := setup(tctx)

		test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilTestEnds()

		// Give the reader a different journal store so we know it isn't reading
		// the getting events from the journal.
		deps.Reader.Journals = &memoryjournal.BinaryStore{}

		events := make(chan Event, 100)
		sub := &Subscriber{
			StreamID: streamID,
			Events:   events,
		}

		test.
			RunInBackground(t, "reader", func(ctx context.Context) error {
				return deps.Reader.Read(ctx, sub)
			}).
			UntilTestEnds()

		env1 := deps.Packer.Pack(MessageE1)
		env2 := deps.Packer.Pack(MessageE2)
		env3 := deps.Packer.Pack(MessageE3)

		go func() {
			// Journal record with a single event.
			if _, err := deps.Supervisor.AppendQueue.Do(
				tctx,
				AppendRequest{
					StreamID: streamID,
					Events: []*envelopepb.Envelope{
						env1,
					},
				},
			); err != nil {
				t.Error(err)
				return
			}

			// Journal record with multiple events.
			if _, err := deps.Supervisor.AppendQueue.Do(
				tctx,
				AppendRequest{
					StreamID: streamID,
					Events: []*envelopepb.Envelope{
						env2,
						env3,
					},
				},
			); err != nil {
				t.Error(err)
				return
			}
		}()

		for offset, env := range []*envelopepb.Envelope{env1, env2, env3} {
			test.ExpectChannelToReceive(
				tctx,
				events,
				Event{
					StreamID: streamID,
					Offset:   Offset(offset),
					Envelope: env,
				},
			)
		}
	})

	t.Run("it reverts to the journal if it cannot keep up with the event stream", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()
		tctx := test.WithContext(t)
		deps := setup(tctx)

		test.
			RunInBackground(t, "supervisor", deps.Supervisor.Run).
			UntilTestEnds()

		events := make(chan Event) // note: unbuffered
		sub := &Subscriber{
			StreamID: streamID,
			Events:   events,
		}

		test.
			RunInBackground(t, "reader", func(ctx context.Context) error {
				return deps.Reader.Read(ctx, sub)
			}).
			UntilTestEnds()

		envelopes := []*envelopepb.Envelope{
			deps.Packer.Pack(MessageE1),
			deps.Packer.Pack(MessageE2),
			deps.Packer.Pack(MessageE3),
		}

		go func() {
			for _, env := range envelopes {
				_, err := deps.Supervisor.AppendQueue.Do(
					tctx,
					AppendRequest{
						StreamID: streamID,
						Events:   []*envelopepb.Envelope{env},
					},
				)
				if err != nil {
					t.Error(err)
					return
				}
			}
		}()

		for offset, env := range envelopes {
			time.Sleep(500 * time.Microsecond) // make the consumer slow
			test.ExpectChannelToReceive(
				tctx,
				events,
				Event{
					StreamID: streamID,
					Offset:   Offset(offset),
					Envelope: env,
				},
			)
		}
	})

	t.Run("it does not duplicate or mis-order events when there are competing supervisors", func(t *testing.T) {
		t.Parallel()

		streamID := uuidpb.Generate()
		tctx := test.WithContext(t)
		deps := setup(tctx)

		// emulate a supervisor running on another node
		remoteSupervisor := &Supervisor{
			Journals: deps.Journals,
			Logger:   deps.Supervisor.Logger.With("supervisor", "remote"),
		}

		deps.Supervisor.Logger = deps.Supervisor.Logger.With("supervisor", "local")

		test.
			RunInBackground(t, "local-supervisor", deps.Supervisor.Run).
			UntilTestEnds()

		test.
			RunInBackground(t, "remote-supervisor", remoteSupervisor.Run).
			UntilTestEnds()

		// use a small buffer, allowing it to revert to reading from the
		// journal as an added sanity check.
		events := make(chan Event, 5)
		sub := &Subscriber{
			StreamID: streamID,
			Events:   events,
		}

		test.
			RunInBackground(t, "reader", func(ctx context.Context) error {
				return deps.Reader.Read(ctx, sub)
			}).
			UntilTestEnds()

		const eventsPerSupervisor = 1

		appendLoop := func(ctx context.Context, s *Supervisor, source string) error {
			handlerID := uuidpb.Generate()

			for n := range eventsPerSupervisor {
				env := deps.Packer.Pack(
					MessageE{Value: n},
					envelope.WithHandler(
						// Abuse the "source handler" field of the envelope to
						// discriminate between events produced by our local and
						// remote supervisors.
						identitypb.New(source, handlerID),
					),
				)

				if _, err := s.AppendQueue.Do(
					ctx,
					AppendRequest{
						StreamID: streamID,
						Events:   []*envelopepb.Envelope{env},
					},
				); err != nil {
					t.Logf("failed to append event #%d: %s", n, err)
					return err
				}
			}

			return nil
		}

		test.
			RunInBackground(t, "local-append-loop", func(ctx context.Context) error {
				return appendLoop(ctx, deps.Supervisor, "local")
			}).
			BeforeTestEnds()

		test.
			RunInBackground(t, "remote-append-loop", func(ctx context.Context) error {
				return appendLoop(ctx, remoteSupervisor, "remote")
			}).
			BeforeTestEnds()

		nextLocal := 0
		nextRemote := 0

		for offset := Offset(0); offset < eventsPerSupervisor*2; offset++ {
			t.Logf("waiting for event at offset %d", offset)

			select {
			case <-tctx.Done():
				t.Fatal(tctx.Err())

			case e := <-events:
				if e.Offset != offset {
					t.Fatalf("unexpected offset: got %d, want %d", e.Offset, offset)
				}

				next := &nextLocal
				if e.Envelope.SourceHandler.Name == "remote" {
					next = &nextRemote
				}

				m, err := deps.Packer.Unpack(e.Envelope)
				if err != nil {
					t.Fatalf("unable to unpack event: %s", err)
				}

				got := m.(MessageE)
				want := MessageE{Value: float64(*next)}

				test.Expect(
					t,
					"unexpected message from "+e.Envelope.SourceHandler.Name+" supervisor",
					got,
					want,
				)

				*next++
				t.Logf("received expected event %q from %q supervisor", e.Envelope.Description, e.Envelope.SourceHandler.Name)
			}
		}

		if nextLocal != eventsPerSupervisor {
			t.Errorf("unexpected number of events from local supervisor: got %d, want %d", nextLocal, eventsPerSupervisor)
		}

		if nextRemote != eventsPerSupervisor {
			t.Errorf("unexpected number of events from remote supervisor: got %d, want %d", nextRemote, eventsPerSupervisor)
		}
	})
}
