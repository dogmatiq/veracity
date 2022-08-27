package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/parcel"
)

type Snapshot struct {
	LastCommandID     string
	UnpublishedEvents []parcel.Parcel
	Root              dogma.AggregateRoot
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
	s.UnpublishedEvents = r.Events
}

type CommandExecutor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Journal         Journal
	Stream          EventStream
	Packer          *parcel.Packer

	snapshot Snapshot
}

func (e *CommandExecutor) Load(ctx context.Context) error {
	var offset uint64

	e.snapshot = Snapshot{
		Root: e.Handler.New(),
	}

	for {
		records, err := e.Journal.Read(
			ctx,
			e.HandlerIdentity.Key,
			e.InstanceID,
			offset,
		)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}

		for _, r := range records {
			offset++
			r.ApplyTo(&e.snapshot)
		}
	}

	if len(e.snapshot.UnpublishedEvents) == 0 {
		return nil
	}

	offset = 0
	for {
		events, err := e.Stream.Read(ctx, offset)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}

		for _, ev := range events {
			if ev.ID() == e.snapshot.UnpublishedEvents[0].ID() {
				e.snapshot.UnpublishedEvents = e.snapshot.UnpublishedEvents[1:]
			}
		}
	}

	for _, ev := range e.snapshot.UnpublishedEvents {
		offset++
		if err := e.Stream.Write(ctx, ev); err != nil {
			return err
		}
	}

	e.snapshot.UnpublishedEvents = nil

	return nil
}

func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	cmd parcel.Parcel,
	ack func(ctx context.Context) error,
) error {
	if len(e.snapshot.UnpublishedEvents) != 0 {
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
