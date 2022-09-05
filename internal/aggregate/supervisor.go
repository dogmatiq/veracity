package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"go.uber.org/zap"
)

type Supervisor struct {
	HandlerIdentity *envelopespec.Identity
	Handler         dogma.AggregateMessageHandler
	Packer          *envelope.Packer
	JournalOpener   journal.Opener[*JournalRecord]
	EventAppender   EventAppender
	Logger          *zap.Logger

	instances map[string]*instance
}

func (s *Supervisor) ExecuteCommand(
	ctx context.Context,
	id string,
	env *envelopespec.Envelope,
) error {
	sup, ok := s.instances[id]
	if !ok {
		j, err := s.JournalOpener.OpenJournal(ctx, id)
		if err != nil {
			return err
		}

		sup = &instance{
			HandlerIdentity: s.HandlerIdentity,
			InstanceID:      id,
			Handler:         s.Handler,
			Packer:          s.Packer,
			Journal:         j,
			EventAppender:   s.EventAppender,
			Logger:          s.Logger.With(zap.String("instance_id", id)),
		}

		if s.instances == nil {
			s.instances = map[string]*instance{}
		}

		s.instances[id] = sup
	}

	return sup.ExecuteCommand(ctx, env)
}
