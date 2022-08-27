package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/parcel"
)

type Snapshot struct {
	LastCommandID string
	PendingEvents []parcel.Parcel
	Root          dogma.AggregateRoot
}

type Record interface {
	ApplyTo(*Snapshot)
}

type commandExecuted struct {
	Command parcel.Parcel
	Events  []parcel.Parcel
}

func (r commandExecuted) ApplyTo(s *Snapshot) {
	s.LastCommandID = r.Command.ID()
	s.PendingEvents = r.Events
}

type CommandExecutor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Loader          *Loader
	Handler         dogma.AggregateMessageHandler
	Journal         Journal
	Stream          EventStream
	Packer          *parcel.Packer

	snapshot Snapshot
}

func (e *CommandExecutor) Load(ctx context.Context) error {
	e.snapshot = Snapshot{
		Root: e.Handler.New(),
	}

	if err := e.Loader.Load(
		ctx,
		e.HandlerIdentity.Key,
		e.InstanceID,
		&e.snapshot,
	); err != nil {
		return err
	}

	for _, ev := range e.snapshot.PendingEvents {
		if err := e.Stream.Write(ctx, ev); err != nil {
			return err
		}
	}

	e.snapshot.PendingEvents = nil

	return nil
}

func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	cmd parcel.Parcel,
	ack func(ctx context.Context) error,
) error {
	if len(e.snapshot.PendingEvents) != 0 {
		panic("cannot execute a new command while there are unpublished events")
	}

	if cmd.ID() == e.snapshot.LastCommandID {
		return ack(ctx)
	}

	s := &scope{
		Executor: e,
		Command:  cmd,
	}
	e.Handler.HandleCommand(e.snapshot.Root, s, cmd.Message)

	if err := e.writeToJournal(
		ctx,
		&commandExecuted{
			Command: cmd,
			Events:  s.Events,
		},
	); err != nil {
		return err
	}

	if err := ack(ctx); err != nil {
		return err
	}

	for _, ev := range s.Events {
		if err := e.Stream.Write(ctx, ev); err != nil {
			return err
		}
	}

	return nil
}

func (e *CommandExecutor) writeToJournal(ctx context.Context, r Record) error {
	if err := e.Journal.Write(
		ctx,
		e.HandlerIdentity.Key,
		e.InstanceID,
		0,
		r,
	); err != nil {
		return err
	}

	r.ApplyTo(&e.snapshot)

	return nil
}
