package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"go.uber.org/zap"
)

// EventAppender is an interface for appending event messages to a stream.
type EventAppender interface {
	Append(ctx context.Context, envelopes ...*envelopespec.Envelope) error
}

type HandlerSupervisor struct {
	HandlerIdentity *envelopespec.Identity
	Handler         dogma.AggregateMessageHandler
	Packer          *envelope.Packer
	EventAppender   EventAppender
	Logger          *zap.Logger

	instances map[string]*InstanceSupervisor
}

func (s *HandlerSupervisor) ExecuteCommand(
	ctx context.Context,
	id string,
	env *envelopespec.Envelope,
) error {
	sup, ok := s.instances[id]
	if !ok {
		sup = &InstanceSupervisor{
			HandlerIdentity: s.HandlerIdentity,
			InstanceID:      id,
			Handler:         s.Handler,
			Packer:          s.Packer,
			EventAppender:   s.EventAppender,
			Logger:          s.Logger.With(zap.String("instance_id", id)),
		}

		if s.instances == nil {
			s.instances = map[string]*InstanceSupervisor{}
		}

		s.instances[id] = sup
	}

	return sup.ExecuteCommand(ctx, env)
}
