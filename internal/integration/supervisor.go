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
	"github.com/dogmatiq/persistencekit/marshaler"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/signaling"
	"google.golang.org/protobuf/proto"
)

// ExecuteRequest is a request to execute a command.
type ExecuteRequest struct {
	Command *envelopepb.Envelope

	// IsFirstAttempt indicates whether or not this is the first request
	// attempting to execute this specific command.
	IsFirstAttempt bool
}

// ExecuteResponse is the response to an ExecuteRequest.
type ExecuteResponse struct {
}

// Supervisor dispatches commands to a specific integration message handler.
type Supervisor struct {
	ExecuteQueue    messaging.ExchangeQueue[ExecuteRequest, ExecuteResponse]
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Journals        journal.BinaryStore
	Keyspaces       kv.BinaryStore
	Packer          *envelope.Packer
	EventRecorder   EventRecorder

	eventStreamID             *uuidpb.UUID
	lowestPossibleEventOffset eventstream.Offset
	journal                   journal.Journal[*journalpb.Record]
	pos                       journal.Position
	handled                   kv.Keyspace[*uuidpb.UUID, bool]
	shutdown                  signaling.Latch
}

// Run starts the supervisor. It runs until ctx is canceled, Shutdown() is
// called, or an error occurs.
func (s *Supervisor) Run(ctx context.Context) error {
	var err error

	journals := journal.NewMarshalingStore(
		s.Journals,
		marshaler.NewProto[*journalpb.Record](),
	)

	s.journal, err = journals.Open(ctx, journalName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer s.journal.Close()

	keyspaces := kv.NewMarshalingStore(
		s.Keyspaces,
		marshaler.NewProto[*uuidpb.UUID](),
		marshaler.Bool,
	)

	s.handled, err = keyspaces.Open(ctx, handledCommandsKeyspaceName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer s.handled.Close()

	return fsm.Start(ctx, s.initState)
}

func (s *Supervisor) initState(ctx context.Context) fsm.Action {
	begin, end, err := s.journal.Bounds(ctx)
	if err != nil {
		return fsm.Fail(err)
	}

	s.pos = begin

	if s.pos == end {
		return fsm.EnterState(s.idleState)
	}

	var (
		unhandled  []*envelopepb.Envelope
		unrecorded []*journalpb.CommandHandled
	)

	// Range over the journal to build a list of pending work consisting of:
	// 	- enqueued but unhandled commands
	// 	- events that have not been recorded to an event stream
	if err := s.journal.Range(
		ctx,
		s.pos,
		func(
			ctx context.Context,
			pos journal.Position,
			record *journalpb.Record,
		) (ok bool, err error) {
			s.pos = pos + 1

			journalpb.Switch_Record_Operation(
				record,
				func(op *journalpb.CommandEnqueued) {
					unhandled = append(unhandled, op.GetCommand())
				},
				func(op *journalpb.CommandHandled) {
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
				func(op *journalpb.EventsAppendedToStream) {
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
		if err := s.recordEvents(ctx, op, false); err != nil {
			return fsm.Fail(err)
		}
	}

	for _, cmd := range unhandled {
		if err := s.handleCommand(ctx, cmd); err != nil {
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

func (s *Supervisor) handleCommand(ctx context.Context, cmd *envelopepb.Envelope) error {
	c, err := s.Packer.Unpack(cmd)
	if err != nil {
		return err
	}

	sc := &scope{}

	if err := s.Handler.HandleCommand(ctx, sc, c); err != nil {
		return err
	}

	if err := s.handled.Set(ctx, cmd.GetMessageId(), true); err != nil {
		return err
	}

	var envs []*envelopepb.Envelope
	for _, ev := range sc.evs {
		envs = append(envs, s.Packer.Pack(ev, envelope.WithCause(cmd), envelope.WithHandler(s.HandlerIdentity)))
	}

	if s.eventStreamID == nil {
		s.eventStreamID, s.lowestPossibleEventOffset, err = s.EventRecorder.SelectEventStream(ctx)
		if err != nil {
			return err
		}
	}

	op := &journalpb.CommandHandled{
		CommandId:                 cmd.GetMessageId(),
		Events:                    envs,
		EventStreamId:             s.eventStreamID,
		LowestPossibleEventOffset: uint64(s.lowestPossibleEventOffset),
	}

	if err := s.journal.Append(
		ctx,
		s.pos,
		journalpb.
			NewRecordBuilder().
			WithCommandHandled(op).
			Build(),
	); err != nil {
		return err
	}
	s.pos++

	return s.recordEvents(ctx, op, true)
}

func (s *Supervisor) recordEvents(
	ctx context.Context,
	op *journalpb.CommandHandled,
	isFirstAttempt bool,
) error {
	if len(op.GetEvents()) == 0 {
		return nil
	}

	res, err := s.EventRecorder.AppendEvents(
		ctx,
		eventstream.AppendRequest{
			StreamID: op.GetEventStreamId(),
			Events:   op.GetEvents(),
			// TODO rename GetStreamOffset in proto
			LowestPossibleOffset: eventstream.Offset(op.GetLowestPossibleEventOffset()),
			IsFirstAttempt:       isFirstAttempt,
		},
	)
	if err != nil {
		return err
	}

	if s.eventStreamID == op.GetEventStreamId() {
		s.lowestPossibleEventOffset = res.EndOffset
	}

	if err := s.journal.Append(
		ctx,
		s.pos,
		journalpb.
			NewRecordBuilder().
			WithEventsAppendedToStream(
				&journalpb.EventsAppendedToStream{
					CommandId: op.GetCommandId(),
				},
			).
			Build(),
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
		alreadyHandled, err := s.handled.Has(ctx, ex.Request.Command.GetMessageId())
		if err != nil {
			ex.Err(err)
			return fsm.Fail(err)
		}
		if alreadyHandled {
			ex.Ok(ExecuteResponse{})
			return fsm.EnterState(s.idleState)
		}
	}

	if err := s.journal.Append(
		ctx,
		s.pos,
		journalpb.
			NewRecordBuilder().
			WithCommandEnqueued(
				&journalpb.CommandEnqueued{
					Command: ex.Request.Command,
				},
			).
			Build(),
	); err != nil {
		ex.Err(err)
		return fsm.Fail(err)
	}

	s.pos++
	ex.Ok(ExecuteResponse{})

	if err := s.handleCommand(ctx, ex.Request.Command); err != nil {
		return fsm.Fail(err)
	}

	return fsm.EnterState(s.idleState)
}
