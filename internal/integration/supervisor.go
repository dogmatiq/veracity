package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/protobuf/protojournal"
	"github.com/dogmatiq/veracity/persistence/journal"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

type Supervisor struct {
	EnqueueCommand  <-chan *EnqueueCommandExchange
	Handler         dogma.IntegrationMessageHandler
	HandlerIdentity *identitypb.Identity
	Journals        journal.Store
	Packer          *envelope.Packer
	EventRecorder   EventRecorder

	pos journal.Position
}

func (s *Supervisor) Run(ctx context.Context) error {
	j, err := s.Journals.Open(ctx, JournalName(s.HandlerIdentity.Key))
	if err != nil {
		return err
	}
	defer j.Close()
	s.pos = 0
	pendingCmds := []*envelopepb.Envelope{}
	pendingEvents := [][]*envelopepb.Envelope{}
	handledCmds := uuidpb.Set{}

	if err := protojournal.Range(
		ctx,
		j,
		0,
		func(
			ctx context.Context,
			pos journal.Position,
			record *journalpb.Record,
		) (ok bool, err error) {
			s.pos = pos + 1

			if rec := record.GetCommandEnqueued(); rec != nil {
				cmd := rec.GetCommand()
				pendingCmds = append(pendingCmds, cmd)
				return true, nil
			}

			if rec := record.GetCommandHandled(); rec != nil {
				// TODO: add to projection...
				cmdID := rec.GetCommandId()
				handledCmds.Add(cmdID)

				for i, cmd := range pendingCmds {
					if proto.Equal(cmd.GetMessageId(), cmdID) {
						pendingCmds = slices.Delete(pendingCmds, i, i+1)
						break
					}
				}
				if len(rec.GetEvents()) > 0 {
					pendingEvents = append(pendingEvents, rec.GetEvents())
				}
				return true, nil
			}

			if rec := record.GetEventsAppendedToStream(); rec != nil {
				for i, events := range pendingEvents {
					if proto.Equal(events[0].GetCausationId(), rec.GetCommandId()) {
						pendingEvents = slices.Delete(pendingEvents, i, i+1)
						break
					}
				}
			}

			return true, nil
		},
	); err != nil {
		return err
	}

	for _, events := range pendingEvents {
		if err := s.recordEvents(ctx, j, events); err != nil {
			return err
		}
	}

	for _, cmd := range pendingCmds {
		if err := s.handleCommand(ctx, cmd, j); err != nil {
			return err
		}
		handledCmds.Add(cmd.GetMessageId())
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ex := <-s.EnqueueCommand:
			if handledCmds.Has(ex.Command.GetMessageId()) {
				ex.Done <- nil
				break
			}
			rec := &journalpb.Record{
				OneOf: &journalpb.Record_CommandEnqueued{
					CommandEnqueued: &journalpb.CommandEnqueued{
						Command: ex.Command,
					},
				},
			}

			if err := protojournal.Append(ctx, j, s.pos, rec); err != nil {
				ex.Done <- err
				return err
			}

			s.pos++
			ex.Done <- nil

			if err := s.handleCommand(ctx, ex.Command, j); err != nil {
				return err
			}

			handledCmds.Add(ex.Command.GetMessageId())
		}
	}
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

	return s.recordEvents(ctx, j, envs)
}

func (s *Supervisor) recordEvents(
	ctx context.Context,
	j journal.Journal,
	envs []*envelopepb.Envelope,
) error {
	if len(envs) == 0 {
		return nil
	}

	if err := s.EventRecorder.RecordEvents(envs); err != nil {
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
