package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
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
	j, err := s.Journals.Open(ctx, "integration/"+s.HandlerIdentity.Key.AsString())
	if err != nil {
		return err
	}
	defer j.Close()
	s.pos = 0
	pendingCmds := []*envelopepb.Envelope{}

	if err := protojournal.RangeAll(
		ctx,
		j,
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
				for i, cmd := range pendingCmds {
					if proto.Equal(cmd.GetMessageId(), rec.GetCommandId()) {
						pendingCmds = slices.Delete(pendingCmds, i, i+1)
						break
					}
				}
				return true, nil
			}

			return true, nil
		},
	); err != nil {
		return err
	}

	for _, cmd := range pendingCmds {
		if err := s.handleCommand(ctx, cmd, j); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ex := <-s.EnqueueCommand:
			rec := &journalpb.Record{
				OneOf: &journalpb.Record_CommandEnqueued{
					CommandEnqueued: &journalpb.CommandEnqueued{
						Command: ex.Command,
					},
				},
			}

			if err := protojournal.Append(ctx, j, s.pos, rec); err != nil {
				return err
			}

			s.pos++
			close(ex.Done)

			if err := s.handleCommand(ctx, ex.Command, j); err != nil {
				return err
			}
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

	s.EventRecorder.RecordEvents(envs)

	return nil
}
