package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/parcel"
)

type Snapshot struct {
}

// func (s *Snapshot) Apply(r Record) {
// 	for _, ev := range r.Events {
// 		s.Root.ApplyEvent(ev)
// 	}
// }

type Record struct {
	// Command *envelopespec.Envelope
	// Events  []*envelopespec.Envelope
}

type CommandExecutor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Stream          EventStream
	Packer          *parcel.Packer
	Root            dogma.AggregateRoot
	HandleCommand   func(dogma.AggregateRoot, dogma.AggregateCommandScope, dogma.Message)

	scope scope
}

func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	cmd parcel.Parcel,
	ack func(ctx context.Context) error,
) error {
	e.scope.Executor = e
	e.scope.Command = cmd
	e.scope.Events = e.scope.Events[:0]

	e.HandleCommand(e.Root, &e.scope, cmd.Message)

	if err := e.Stream.Write(ctx, e.scope.Events); err != nil {
		return err
	}

	return ack(ctx)
}
