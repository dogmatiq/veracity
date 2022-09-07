package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	envelopespec "github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
)

// EventAppender is an interface for appending event messages to a stream.
type EventAppender interface {
	Append(ctx context.Context, envelopes ...*envelopespec.Envelope) error
}

// CommandExecutor executes commands against aggregate instances of a single
// type.
type CommandExecutor struct {
	HandlerIdentity *envelopespec.Identity
	Handler         dogma.AggregateMessageHandler
	Packer          *envelope.Packer
	JournalOpener   journal.Opener[*JournalRecord]
	EventAppender   EventAppender
	Logger          *zap.Logger

	instances map[string]*instance
}

// ExecuteCommand executes a command against the given aggregate instance.
func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	id string,
	env *envelopespec.Envelope,
) error {
	sup, ok := e.instances[id]
	if !ok {
		j, err := e.JournalOpener.OpenJournal(ctx, id)
		if err != nil {
			return err
		}

		sup = &instance{
			HandlerIdentity: e.HandlerIdentity,
			InstanceID:      id,
			Handler:         e.Handler,
			Packer:          e.Packer,
			Journal:         j,
			EventAppender:   e.EventAppender,
			Logger:          e.Logger.With(zap.String("instance_id", id)),
		}

		if e.instances == nil {
			e.instances = map[string]*instance{}
		}

		e.instances[id] = sup
	}

	return sup.ExecuteCommand(ctx, env)
}
