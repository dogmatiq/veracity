package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/protobuf/protojournal"
	"github.com/dogmatiq/veracity/internal/signaling"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

type ExecuteRequest struct {
	Command *envelopepb.Envelope

	// IsFirstAttempt indicates whether or not this is the first request
	// attempting to execute this specific command.
	IsFirstAttempt bool
}

type ExecuteResponse struct {
}

type Supervisor struct {
	ExecuteQueue    messaging.ExchangeQueue[ExecuteRequest, ExecuteResponse]
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Journals        journal.Store
	Keyspaces       kv.Store
	Packer          *envelope.Packer
	EventRecorder   EventRecorder

	// TODO: add a field to store the event stream offset.
	eventStreamID     *uuidpb.UUID
	eventStreamOffset eventstream.Offset
	journal           journal.Journal
	pos               journal.Position
	handledCmds       kv.Keyspace
	shutdown          signaling.Latch
}

func (s *Supervisor) Run(ctx context.Context) error {
	var err error

	s.journal, err = s.Journals.Open(ctx, JournalName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer s.journal.Close()

	s.handledCmds, err = s.Keyspaces.Open(ctx, HandledCommandsKeyspaceName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer s.journal.Close()

	return fsm.Start(ctx, s.initState)
}

func (s *Supervisor) initState(ctx context.Context) fsm.Action {
	var pendingCmds []*envelopepb.Envelope
	var pendingEvents []*journalpb.CommandHandled
	var err error

	s.pos, _, err = s.journal.Bounds(ctx)
	if err != nil {
		return fsm.Fail(err)
	}

	if err := protojournal.Range(
		ctx,
		s.journal,
		s.pos,
		func(
			ctx context.Context,
			pos journal.Position,
			record *journalpb.Record,
		) (ok bool, err error) {
			s.pos = pos + 1

			record.DispatchOneOf(
				func(rec *journalpb.CommandEnqueued) {
					cmd := rec.GetCommand()
					pendingCmds = append(pendingCmds, cmd)
				},
				func(rec *journalpb.CommandHandled) {
					cmdID := rec.GetCommandId()
					for i, cmd := range pendingCmds {
						if proto.Equal(cmd.GetMessageId(), cmdID) {
							pendingCmds = slices.Delete(pendingCmds, i, i+1)
							break
						}
					}
					if len(rec.GetEvents()) > 0 {
						pendingEvents = append(pendingEvents, rec)
					}
				},
				func(*journalpb.CommandHandlerFailed) {
					// ignore
				},
				func(rec *journalpb.EventsAppendedToStream) {
					for i, candidate := range pendingEvents {
						if proto.Equal(candidate.GetCommandId(), rec.GetCommandId()) {
							pendingEvents = slices.Delete(pendingEvents, i, i+1)
							break
						}
					}
				},
				func() {
					panic("unrecognized record type")
				},
			)

			return true, err
		},
	); err != nil {
		return fsm.Fail(err)
	}

	for _, rec := range pendingEvents {
		if err := s.recordEvents(ctx, s.journal, rec, false); err != nil {
			return fsm.Fail(err)
		}
	}

	for _, cmd := range pendingCmds {
		if err := s.handleCommand(ctx, cmd, s.journal); err != nil {
			return fsm.Fail(err)
		}
	}

	if err := s.journal.Truncate(ctx, s.pos); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idleState)
}

func (s *Supervisor) idleState(ctx context.Context) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()

	case <-s.shutdown.Signaled():
		return fsm.Stop()

	case ex := <-s.ExecuteQueue.Recv():
		return fsm.With(ex).EnterState(s.handleCommandState)
	}
}

// Shutdown stops the supervisor when it next becomes idle.
func (s *Supervisor) Shutdown() {
	s.shutdown.Signal()
}

func (s *Supervisor) handleCommand(ctx context.Context, cmd *envelopepb.Envelope, j journal.Journal) error {
	c, err := s.Packer.Unpack(cmd)
	if err != nil {
		return err
	}

	sc := &scope{}

	if handlerErr := s.Handler.HandleCommand(ctx, sc, c); handlerErr != nil {
		if err := protojournal.Append(ctx, j, s.pos, &journalpb.Record{OneOf: &journalpb.Record_CommandHandlerFailed{
			CommandHandlerFailed: &journalpb.CommandHandlerFailed{
				CommandId: cmd.GetMessageId(),
				Error:     handlerErr.Error(),
			},
		}}); err != nil {
			return err
		}

		s.pos++

		return handlerErr
	}

	if err := s.handledCmds.Set(
		ctx,
		cmd.GetMessageId().AsBytes(),
		[]byte{1},
	); err != nil {
		return err
	}

	var envs []*envelopepb.Envelope
	for _, ev := range sc.evs {
		envs = append(envs, s.Packer.Pack(ev, envelope.WithCause(cmd), envelope.WithHandler(s.HandlerIdentity)))
	}

	if s.eventStreamID == nil {
		s.eventStreamID, s.eventStreamOffset, err = s.EventRecorder.SelectEventStream(ctx)
		if err != nil {
			return err
		}
	}

	rec := &journalpb.CommandHandled{
		CommandId:    cmd.GetMessageId(),
		Events:       envs,
		StreamId:     s.eventStreamID,
		StreamOffset: uint64(s.eventStreamOffset),
	}

	if err := protojournal.Append(ctx, j, s.pos, &journalpb.Record{
		OneOf: &journalpb.Record_CommandHandled{
			CommandHandled: rec,
		},
	}); err != nil {
		return err
	}
	s.pos++

	return s.recordEvents(ctx, j, rec, true)
}

func (s *Supervisor) recordEvents(
	ctx context.Context,
	j journal.Journal,
	rec *journalpb.CommandHandled,
	isFirstAttempt bool,
) error {
	if len(rec.GetEvents()) == 0 {
		return nil
	}

	if err := s.EventRecorder.AppendEvents(
		ctx,
		eventstream.AppendRequest{
			StreamID:             rec.GetStreamId(),
			Events:               rec.GetEvents(),
			LowestPossibleOffset: eventstream.Offset(rec.GetStreamOffset()),
			IsFirstAttempt:       isFirstAttempt,
		},
	); err != nil {
		return err
	}

	if err := protojournal.Append(
		ctx,
		j,
		s.pos,
		&journalpb.Record{
			OneOf: &journalpb.Record_EventsAppendedToStream{
				EventsAppendedToStream: &journalpb.EventsAppendedToStream{
					CommandId: rec.GetCommandId(),
				},
			},
		},
	); err != nil {
		return err
	}
	s.pos++

	return nil
}

// handleCommandState executes commands.
func (s *Supervisor) handleCommandState(
	ctx context.Context,
	ex messaging.Exchange[ExecuteRequest, ExecuteResponse],
) fsm.Action {
	if !ex.Request.IsFirstAttempt {
		alreadyHandled, err := s.handledCmds.Has(
			ctx,
			ex.Request.Command.GetMessageId().AsBytes(),
		)
		if err != nil {
			ex.Err(err)
			return fsm.Fail(err)
		}
		if alreadyHandled {
			ex.Ok(ExecuteResponse{})
			return fsm.EnterState(s.idleState)
		}
	}

	rec := &journalpb.Record{
		OneOf: &journalpb.Record_CommandEnqueued{
			CommandEnqueued: &journalpb.CommandEnqueued{
				Command: ex.Request.Command,
			},
		},
	}

	if err := protojournal.Append(ctx, s.journal, s.pos, rec); err != nil {
		ex.Err(err)
		return fsm.Fail(err)
	}

	s.pos++
	ex.Ok(ExecuteResponse{})

	if err := s.handleCommand(ctx, ex.Request.Command, s.journal); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idleState)
}

// JournalName returns the name of the journal that contains the state
// of the integration handler with the given key.
func JournalName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString()
}

// HandledCommandsKeyspaceName returns the name of the keyspace that contains
// the set of handled command IDs.
func HandledCommandsKeyspaceName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString() + ":handled-commands"
}
