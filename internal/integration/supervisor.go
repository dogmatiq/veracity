package integration

import (
	"context"
	"slices"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationjournal"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationkv"
	"github.com/dogmatiq/veracity/internal/messaging/ackqueue"
	"github.com/dogmatiq/veracity/internal/signaling"
	"google.golang.org/protobuf/proto"
)

// Supervisor dispatches commands to a specific integration message handler.
type Supervisor struct {
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Commands        ackqueue.Queue[*envelopepb.Envelope]
	Journals        journal.BinaryStore
	Keyspaces       kv.BinaryStore
	Packer          *envelopepb.Packer
	Events          EventRecorder

	journal         journal.Journal[*integrationjournal.Record]
	journalPos      journal.Position
	eventStreamID   *uuidpb.UUID
	eventOffsetHint eventstream.Offset
	handled         kv.Keyspace[*uuidpb.UUID, bool]
	shutdown        signaling.Latch
}

// Run starts the supervisor.
//
// It runs until ctx is canceled, Shutdown() is called, or an error occurs.
func (s *Supervisor) Run(ctx context.Context) error {
	var err error

	s.journal, err = integrationjournal.Open(ctx, s.Journals, s.HandlerIdentity.Key)
	if err != nil {
		return err
	}
	defer s.journal.Close()

	s.handled, err = integrationkv.OpenHandledCommands(ctx, s.Keyspaces, s.HandlerIdentity.Key)
	if err != nil {
		return err
	}
	defer s.handled.Close()

	return fsm.Start(ctx, s.recover)
}

// Shutdown instructs the supervisor to shutdown when it next enters the idle
// state.
func (s *Supervisor) Shutdown() {
	s.shutdown.Signal()
}

// recover discovers any pending work in the journal, and completes it before
// entering the idle state.
func (s *Supervisor) recover(ctx context.Context) fsm.Action {
	bounds, err := s.journal.Bounds(ctx)
	if err != nil {
		return fsm.Fail(err)
	}

	s.journalPos = bounds.Begin

	// There's nothing in the journal, so nothing to recover from.
	if bounds.IsEmpty() {
		return fsm.EnterState(s.idle)
	}

	// Otherwise, we may have unhandled commands, or events that have not yet
	// been recorded to an event stream.
	var (
		unhandled  []*envelopepb.Envelope
		unrecorded []*integrationjournal.CommandHandled
	)

	// Range over the journal to build a list of pending work.
	if err := s.journal.Range(
		ctx,
		s.journalPos,
		func(
			_ context.Context,
			pos journal.Position,
			record *integrationjournal.Record,
		) (ok bool, err error) {
			s.journalPos = pos + 1

			integrationjournal.MustSwitch_Record_Operation(
				record,
				func(op *integrationjournal.CommandEnqueued) {
					unhandled = append(unhandled, op.GetCommand())
				},
				func(op *integrationjournal.CommandHandled) {
					for i, env := range unhandled {
						if proto.Equal(env.GetMessageId(), op.GetCommandId()) {
							unhandled = slices.Delete(unhandled, i, i+1)
							break
						}
					}
					if len(op.GetEvents()) > 0 {
						unrecorded = append(unrecorded, op)
					}
				},
				func(op *integrationjournal.EventsAppendedToStream) {
					for i, rec := range unrecorded {
						if proto.Equal(rec.GetCommandId(), op.GetCommandId()) {
							unrecorded = slices.Delete(unrecorded, i, i+1)
							break
						}
					}
				},
			)

			return true, err
		},
	); err != nil {
		return fsm.Fail(err)
	}

	for _, op := range unrecorded {
		if err := s.recordEvents(ctx, op); err != nil {
			return fsm.Fail(err)
		}
	}

	for _, cmd := range unhandled {
		if err := s.handleCommand(ctx, cmd); err != nil {
			return fsm.Fail(err)
		}
	}

	// We've now confirmed that there is no pending work in the journal, so the
	// entire thing can be truncated.
	if err := s.journal.Truncate(ctx, s.journalPos); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idle)
}

// idle waits for a command request to be received, or the shutdown signal.
func (s *Supervisor) idle(ctx context.Context) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()

	case <-s.shutdown.Signaled():
		return fsm.Stop()

	case req := <-s.Commands.Recv():
		return fsm.With(req).EnterState(s.accept)
	}
}

// accept processes a command request by persisting it to the journal, then
// handling the command.
func (s *Supervisor) accept(
	ctx context.Context,
	req ackqueue.Request[*envelopepb.Envelope],
) fsm.Action {
	// TODO: there are optimizations to be made here (i.e. in-memory list of
	// recent commands, bloom filter, etc).
	alreadyHandled, err := s.handled.Has(ctx, req.Value.GetMessageId())
	if err != nil {
		req.Nack(err)
		return fsm.Fail(err)
	}

	if alreadyHandled {
		req.Ack()
		return fsm.EnterState(s.idle)
	}

	if err := s.journal.Append(
		ctx,
		s.journalPos,
		integrationjournal.
			NewRecordBuilder().
			WithCommandEnqueued(
				&integrationjournal.CommandEnqueued{
					Command: req.Value,
				},
			).
			Build(),
	); err != nil {
		req.Nack(err)
		return fsm.Fail(err)
	}

	s.journalPos++
	req.Ack()

	if err := s.handleCommand(ctx, req.Value); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idle)
}

// handleCommand dispatches a command to the [dogma.IntegrationMessageHandler]
// and persists the result to the journal.
func (s *Supervisor) handleCommand(ctx context.Context, env *envelopepb.Envelope) error {
	cmd, err := s.Packer.Unpack(env)
	if err != nil {
		return err
	}

	sc := &scope{
		packer:  s.Packer,
		handler: s.HandlerIdentity,
		command: env,
	}

	if err := s.Handler.HandleCommand(
		ctx,
		sc,
		cmd.(dogma.Command),
	); err != nil {
		return err
	}

	// TODO: Is this in the right place? Should we do this before or after
	// persisting the result to the journal.
	if err := s.handled.Set(ctx, env.GetMessageId(), true); err != nil {
		return err
	}

	op := &integrationjournal.CommandHandled{
		CommandId: env.GetMessageId(),
		Events:    sc.events,
	}

	if len(sc.events) != 0 {
		if s.eventStreamID == nil {
			// Determine which event stream to which the events should be
			// appended, then use this stream from now on.
			s.eventStreamID, s.eventOffsetHint, err = s.Events.SelectEventStream(ctx)
			if err != nil {
				return err
			}
		}

		op.EventStreamId = s.eventStreamID
		op.OffsetHint = uint64(s.eventOffsetHint)
	}

	if err := s.journal.Append(
		ctx,
		s.journalPos,
		integrationjournal.
			NewRecordBuilder().
			WithCommandHandled(op).
			Build(),
	); err != nil {
		return err
	}
	s.journalPos++

	return s.recordEvents(ctx, op)
}

// recordEvents appends the events produced by a command to their target event
// stream.
func (s *Supervisor) recordEvents(
	ctx context.Context,
	op *integrationjournal.CommandHandled,
) error {
	if len(op.GetEvents()) == 0 {
		return nil
	}

	res, err := s.Events.AppendEvents(
		ctx,
		eventstream.AppendRequest{
			StreamID:   op.GetEventStreamId(),
			Events:     op.GetEvents(),
			OffsetHint: eventstream.Offset(op.GetOffsetHint()),
		},
	)
	if err != nil {
		return err
	}

	if s.eventStreamID == op.GetEventStreamId() {
		s.eventOffsetHint = res.EndOffset
	}

	if err := s.journal.Append(
		ctx,
		s.journalPos,
		integrationjournal.
			NewRecordBuilder().
			WithEventsAppendedToStream(
				&integrationjournal.EventsAppendedToStream{
					CommandId: op.GetCommandId(),
				},
			).
			Build(),
	); err != nil {
		return err
	}
	s.journalPos++

	return nil
}
