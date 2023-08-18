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
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

type ExecuteRequest struct {
	// TODO: IsFirstAttempt
	Command *envelopepb.Envelope
}

type ExecuteResponse struct {
}

type Supervisor struct {
	ExecuteQueue    messaging.ExchangeQueue[ExecuteRequest, ExecuteResponse]
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Journals        journal.Store
	Packer          *envelope.Packer
	EventRecorder   EventRecorder

	journal     journal.Journal
	pos         journal.Position
	handledCmds uuidpb.Set
	shutdown    signaling.Latch
}

func (s *Supervisor) Run(ctx context.Context) error {
	var err error
	s.journal, err = s.Journals.Open(ctx, JournalName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer s.journal.Close()

	return fsm.Start(ctx, s.initState)
}

func (s *Supervisor) initState(ctx context.Context) fsm.Action {
	s.pos = 0

	pendingCmds := []*envelopepb.Envelope{}
	pendingEvents := [][]*envelopepb.Envelope{}
	s.handledCmds = uuidpb.Set{}

	if err := protojournal.Range(
		ctx,
		s.journal,
		0,
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
					// TODO: add to projection...
					cmdID := rec.GetCommandId()
					s.handledCmds.Add(cmdID)

					for i, cmd := range pendingCmds {
						if proto.Equal(cmd.GetMessageId(), cmdID) {
							pendingCmds = slices.Delete(pendingCmds, i, i+1)
							break
						}
					}
					if len(rec.GetEvents()) > 0 {
						pendingEvents = append(pendingEvents, rec.GetEvents())
					}
				},
				func(*journalpb.CommandHandlerFailed) {
					// ignore
				},
				func(*journalpb.EventStreamSelected) {
					// ignore
				},
				func(rec *journalpb.EventsAppendedToStream) {
					for i, events := range pendingEvents {
						if proto.Equal(events[0].GetCausationId(), rec.GetCommandId()) {
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

	for _, events := range pendingEvents {
		if err := s.recordEvents(ctx, s.journal, events, false); err != nil {
			return fsm.Fail(err)
		}
	}

	for _, cmd := range pendingCmds {
		if err := s.handleCommand(ctx, cmd, s.journal); err != nil {
			return fsm.Fail(err)
		}
		s.handledCmds.Add(cmd.GetMessageId())
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

	var envs []*envelopepb.Envelope
	for _, ev := range sc.evs {
		envs = append(envs, s.Packer.Pack(ev, envelope.WithCause(cmd), envelope.WithHandler(s.HandlerIdentity)))
	}

	if err := protojournal.Append(ctx, j, s.pos, &journalpb.Record{
		OneOf: &journalpb.Record_CommandHandled{
			CommandHandled: &journalpb.CommandHandled{
				CommandId: cmd.GetMessageId(),
				Events:    envs,
			},
		},
	}); err != nil {
		return err
	}

	s.pos++

	return s.recordEvents(ctx, j, envs, true)
}

func (s *Supervisor) recordEvents(
	ctx context.Context,
	j journal.Journal,
	envs []*envelopepb.Envelope,
	isFirstAttempt bool,
) error {
	if len(envs) == 0 {
		return nil
	}

	if err := s.EventRecorder.AppendEvents(
		ctx,
		eventstream.AppendRequest{
			// TODO: set minimum offset
			Events:         envs,
			IsFirstAttempt: isFirstAttempt,
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
					CommandId: envs[0].GetCausationId(),
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
	if s.handledCmds.Has(ex.Request.Command.GetMessageId()) {
		ex.Ok(ExecuteResponse{})
		return fsm.EnterState(s.idleState)
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

	s.handledCmds.Add(ex.Request.Command.GetMessageId())
	return fsm.EnterState(s.idleState)
}
