package integration

import (
	"context"
	"slices"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/set"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationjournal"
	"github.com/dogmatiq/veracity/internal/integration/internal/integrationset"
	"github.com/dogmatiq/veracity/internal/messaging/ackqueue"
	"github.com/dogmatiq/veracity/internal/signaling"
	"github.com/dogmatiq/veracity/internal/telemetry"
)

// Supervisor dispatches commands to a specific integration message handler.
type Supervisor struct {
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Commands        ackqueue.Queue[*envelopepb.Envelope]
	Journals        journal.BinaryStore
	Sets            set.BinaryStore
	Packer          *envelopepb.Packer
	Events          EventRecorder
	Telemetry       *telemetry.Provider

	journal          journal.Journal[*integrationjournal.Record]
	journalPos       journal.Position
	eventStreamID    *uuidpb.UUID
	eventOffsetHint  eventstream.Offset
	acceptedCommands set.Set[*uuidpb.UUID]
	shutdownLatch    signaling.Latch
	telemetry        *telemetry.Recorder
}

// Run starts the supervisor.
//
// It runs until ctx is canceled, Shutdown() is called, or an error occurs.
func (s *Supervisor) Run(ctx context.Context) error {
	s.telemetry = s.Telemetry.Recorder(
		telemetry.UUID("handler.key", s.HandlerIdentity.Key),
		telemetry.String("handler.name", s.HandlerIdentity.Name),
	)

	var err error

	s.journal, err = integrationjournal.Open(ctx, s.Journals, s.HandlerIdentity.Key)
	if err != nil {
		return err
	}
	defer s.journal.Close()

	s.acceptedCommands, err = integrationset.OpenAcceptedCommandsSet(ctx, s.Sets, s.HandlerIdentity.Key)
	if err != nil {
		return err
	}
	defer s.acceptedCommands.Close()

	return fsm.Start(ctx, s.recover)
}

// Shutdown instructs the supervisor to shutdown when it next enters the idle
// state.
func (s *Supervisor) Shutdown() {
	s.shutdownLatch.Signal()
}

// recover discovers any pending work in the journal, and completes it before
// entering the `idle state`.
func (s *Supervisor) recover(ctx context.Context) fsm.Action {
	ctx, span := s.telemetry.StartSpan(ctx, "integration.recover")
	defer span.End()

	// We may have unhandled commands or events that have not yet been recorded
	// to an event stream.
	var (
		// acceptedCommands is the list of operations representing commands that
		// were accepted by the supervisor but have either not yet been passed
		// to the handler, or the result of doing so has not been recorded to
		// the journal and therefore they must be retried.
		acceptedCommands []*integrationjournal.CommandAccepted

		// handledCommands is the list of commands that have been handled
		// successfully, but may have events that have not yet been appended to
		// an event stream.
		handledCommands []*integrationjournal.CommandHandled
	)

	s.telemetry.Info(ctx, "integration.recover", "recovering pending operations from journal")

	bounds, err := s.journal.Bounds(ctx)
	if err != nil {
		return fsm.Fail(err)
	}

	s.journalPos = bounds.Begin

	// Range over the journal to build a list of pending operations.
	if !bounds.IsEmpty() {
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
					func(op *integrationjournal.CommandAccepted) {
						acceptedCommands = append(acceptedCommands, op)
					},
					func(op *integrationjournal.CommandHandled) {
						for i, x := range acceptedCommands {
							if x.GetCommand().GetMessageId().Equal(op.GetCommandId()) {
								acceptedCommands = slices.Delete(acceptedCommands, i, i+1)
								break
							}
						}
						handledCommands = append(handledCommands, op)
					},
					func(op *integrationjournal.EventsAppendedToStream) {
						for i, x := range handledCommands {
							if x.GetCommandId().Equal(op.GetCommandId()) {
								handledCommands = slices.Delete(handledCommands, i, i+1)
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
	}

	span.SetAttributes(
		telemetry.Int("unhandled_commands", len(acceptedCommands)),
		telemetry.Int("unappended_events", len(handledCommands)),
	)

	if len(acceptedCommands) == 0 && len(handledCommands) == 0 {
		s.telemetry.Info(ctx, "integration.recover.ok", "no pending operations found in journal")
	} else {
		if len(handledCommands) != 0 {
			s.telemetry.Info(
				ctx,
				"integration.recover.unappended_event_found",
				"journal contains events that have not yet been appended to an event stream",
			)

			for _, op := range handledCommands {
				if err := s.appendEvents(ctx, op); err != nil {
					return fsm.Fail(err)
				}
			}
		}

		if len(acceptedCommands) != 0 {
			s.telemetry.Info(
				ctx,
				"integration.recover.unhandled_command_found",
				"journal contains commands that have been accepted but not yet handled",
			)

			for _, op := range acceptedCommands {
				if err := s.handleCommand(ctx, op.GetCommand()); err != nil {
					return fsm.Fail(err)
				}
			}
		}

		s.telemetry.Info(ctx, "integration.recover.ok", "all pending operations in journal are now complete")
	}

	// We've now confirmed that there is no pending work remaining in the
	// journal, so the entire thing can be truncated.
	if err := s.journal.Truncate(ctx, s.journalPos); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idle)
}

// idle waits for a command request to be received, or the shutdown signal.
func (s *Supervisor) idle(ctx context.Context) fsm.Action {
	// TODO: should we truncate the journal if we're idle for a certain period
	// of time and/or if it's over a certain size? Otherwise, it only happens on
	// startup/recovery.

	select {
	case <-ctx.Done():
		return fsm.Stop()

	case <-s.shutdownLatch.Signaled():
		return fsm.Stop()

	case req := <-s.Commands.Recv():
		return fsm.With(req).EnterState(s.acceptCommand)
	}
}

// accept processes a command request by persisting it to the journal, then
// handling the command.
func (s *Supervisor) acceptCommand(
	ctx context.Context,
	req ackqueue.Request[*envelopepb.Envelope],
) fsm.Action {
	env := req.Value

	ctx, span := s.telemetry.StartSpan(
		ctx,
		"integration.accept_command",
		telemetry.UUID("command.message_id", env.GetMessageId()),
		telemetry.UUID("command.causation_id", env.GetMessageId()),
		telemetry.UUID("command.correlation_id", env.GetCorrelationId()),
		telemetry.String("command.media_type", env.GetMediaType()),
		telemetry.String("command.description", env.GetDescription()),
	)
	defer span.End()

	// Do not accept the command if it has already been accepted in the past.
	// This check provides command-level idempotency, even if the journal has
	// been truncated.
	//
	// TODO: there are optimizations to be made here (i.e. in-memory list of
	// recent commands, bloom filter, etc).
	alreadyAccepted, err := s.acceptedCommands.Has(ctx, env.GetMessageId())
	if err != nil {
		req.Nack()
		return fsm.Fail(err)
	}

	if alreadyAccepted {
		req.Ack()

		s.telemetry.Info(
			ctx,
			"integration.accept_command.ok",
			"acknowledged and ignored command that has already been accepted for handling",
		)

		return fsm.EnterState(s.idle)
	}

	if err := s.journal.Append(
		ctx,
		s.journalPos,
		integrationjournal.
			NewRecordBuilder().
			WithCommandAccepted(
				&integrationjournal.CommandAccepted{
					Command: env,
				},
			).
			Build(),
	); err != nil {
		s.telemetry.Error(ctx, "integration.accept_command.error", err)
		req.Nack()
		return fsm.Fail(err)
	}

	s.journalPos++
	req.Ack()

	if err := s.handleCommand(ctx, env); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idle)
}

// handleCommand dispatches a command to the [dogma.IntegrationMessageHandler]
// and persists the result to the journal.
func (s *Supervisor) handleCommand(ctx context.Context, env *envelopepb.Envelope) error {
	ctx, span := s.telemetry.StartSpan(
		ctx,
		"integration.handle_command",
		telemetry.UUID("command.message_id", env.GetMessageId()),
		telemetry.UUID("command.causation_id", env.GetMessageId()),
		telemetry.UUID("command.correlation_id", env.GetCorrelationId()),
		telemetry.String("command.media_type", env.GetMediaType()),
		telemetry.String("command.description", env.GetDescription()),
	)
	defer span.End()

	if err := s.acceptedCommands.Add(ctx, env.GetMessageId()); err != nil {
		s.telemetry.Error(ctx, "integration.handle_command.error", err)
		return err
	}

	cmd, err := s.Packer.Unpack(env)
	if err != nil {
		s.telemetry.Error(ctx, "integration.handle_command.error", err)
		return err
	}

	sc := &scope{
		ctx:       ctx,
		packer:    s.Packer,
		handler:   s.HandlerIdentity,
		command:   env,
		telemetry: s.telemetry,
	}

	if err := s.Handler.HandleCommand(
		ctx,
		sc,
		cmd.(dogma.Command),
	); err != nil {
		s.telemetry.Error(ctx, "integration.handle_command.error", err)
		return err
	}

	op := &integrationjournal.CommandHandled{
		CommandId: env.GetMessageId(),
		Events:    sc.events,
	}

	if len(sc.events) != 0 {
		// Determine the event stream to which the events should be appended,
		// then use this stream from now on.
		if s.eventStreamID == nil {
			s.eventStreamID, s.eventOffsetHint, err = s.Events.SelectEventStream(ctx)
			if err != nil {
				s.telemetry.Error(ctx, "integration.handle_command.error", err)
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
		s.telemetry.Error(ctx, "integration.handle_command.error", err)
		return err
	}
	s.journalPos++

	s.telemetry.Info(ctx, "integration.handle_command.ok", "command handled successfully")

	return s.appendEvents(ctx, op)
}

// appendEvents appends the events produced by a command to their target event
// stream.
func (s *Supervisor) appendEvents(
	ctx context.Context,
	op *integrationjournal.CommandHandled,
) error {
	if len(op.GetEvents()) == 0 {
		return nil
	}

	ctx, span := s.telemetry.StartSpan(ctx, "integration.append_events")
	defer span.End()

	res, err := s.Events.AppendEvents(
		ctx,
		eventstream.AppendRequest{
			StreamID:   op.GetEventStreamId(),
			Events:     op.GetEvents(),
			OffsetHint: eventstream.Offset(op.GetOffsetHint()),
		},
	)
	if err != nil {
		s.telemetry.Error(ctx, "integration.append_events.error", err)
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
		s.telemetry.Error(ctx, "integration.append_events.error", err)
		return err
	}
	s.journalPos++

	s.telemetry.Info(ctx, "integration.append_events.ok", "events appended to event stream successfully")

	return nil
}
