package aggregate

import (
	"context"
	"sync"

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

	once     sync.Once
	requests chan request

	instances map[string]chan request
}

type request struct {
	InstanceID      string
	CommandEnvelope *envelopespec.Envelope
	Response        chan<- error
}

func (e *CommandExecutor) Run(ctx context.Context) error {
	e.once.Do(func() {
		e.requests = make(chan request)
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-e.requests:
			if err := e.executeCommand(ctx, req); err != nil {
				return err
			}
		}
	}
}

func (e *CommandExecutor) executeCommand(ctx context.Context, req request) error {
	requests, ok := e.instances[req.InstanceID]
	if !ok {
		requests = make(chan request)

		go func() {
			j, err := e.JournalOpener.OpenJournal(ctx, req.InstanceID)
			if err != nil {
				panic(err)
			}

			inst := &instance{
				HandlerIdentity: e.HandlerIdentity,
				InstanceID:      req.InstanceID,
				Handler:         e.Handler,
				Packer:          e.Packer,
				Journal:         j,
				EventAppender:   e.EventAppender,
				Logger:          e.Logger.With(zap.String("instance_id", req.InstanceID)),
				Requests:        requests,
			}

			inst.Run(ctx)
		}()

		if e.instances == nil {
			e.instances = map[string]chan request{}
		}

		e.instances[req.InstanceID] = requests
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case requests <- req:
		return nil
	}
}

// ExecuteCommand executes a command against the given aggregate instance.
func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	id string,
	env *envelopespec.Envelope,
) error {
	e.once.Do(func() {
		e.requests = make(chan request)
	})

	response := make(chan error, 1)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.requests <- request{id, env, response}:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-response:
		return err
	}
}
