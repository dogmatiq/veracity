package integration

import (
	"context"
	"errors"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

type Supervisor struct {
	EnqueueCommand <-chan *EnqueueCommandExchange
	Handler        dogma.IntegrationMessageHandler
	HandlerKey     string
	Journals       journal.Store
	Packer         *envelope.Packer
}

func (s *Supervisor) Run(ctx context.Context) error {
	j, err := s.Journals.Open(ctx, "integration/"+s.HandlerKey)
	if err != nil {
		return err
	}
	defer j.Close()

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

			data, err := proto.Marshal(rec)
			if err != nil {
				return err
			}

			ok, err := j.Append(ctx, 0, data)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("optimistic concurrency control failure occurred")
			}

			close(ex.Done)

			c, err := s.Packer.Unpack(ex.Command)
			if err != nil {
				return err
			}

			if err := s.Handler.HandleCommand(ctx, &scope{}, c); err != nil {
				return err
			}
		}
	}
}
