package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/protobuf/protojournal"
	"github.com/dogmatiq/veracity/persistence/journal"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
)

type Supervisor struct {
	EnqueueCommand <-chan *EnqueueCommandExchange
	Handler        dogma.IntegrationMessageHandler
	HandlerKey     string
	Journals       journal.Store
	Packer         *envelope.Packer

	offset uint64
}

func (s *Supervisor) Run(ctx context.Context) error {
	j, err := s.Journals.Open(ctx, "integration/"+s.HandlerKey)
	if err != nil {
		return err
	}
	defer j.Close()
	s.offset = 0
	pendingCmds := []*envelopepb.Envelope{}
	if err := protojournal.RangeAll(
		ctx,
		j,
		func(ctx context.Context, offset uint64, record *journalpb.Record) (ok bool, err error) {
			s.offset = offset + 1
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

			if err := protojournal.Append(ctx, j, s.offset, rec); err != nil {
				return err
			}

			s.offset++
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

	rec := &journalpb.Record{}

	handlerErr := s.Handler.HandleCommand(ctx, &scope{}, c)

	if handlerErr != nil {
		rec.OneOf = &journalpb.Record_CommandHandlerFailed{
			CommandHandlerFailed: &journalpb.CommandHandlerFailed{
				CommandId: cmd.GetMessageId(),
				Error:     handlerErr.Error(),
			},
		}
	} else {
		rec.OneOf = &journalpb.Record_CommandHandled{
			CommandHandled: &journalpb.CommandHandled{
				CommandId: cmd.GetMessageId(),
			},
		}
	}

	if err := protojournal.Append(ctx, j, s.offset, rec); err != nil {
		return err
	}

	s.offset++

	return handlerErr
}
