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
}

func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	cmd parcel.Parcel,
) error {
	s := &scope{
		Command:         cmd,
		HandlerIdentity: e.HandlerIdentity,
		ID:              e.InstanceID,
		Packer:          e.Packer,
		Root:            e.Root,
	}

	e.HandleCommand(e.Root, s, cmd.Message)

	return e.Stream.Write(ctx, s.Events)
}
