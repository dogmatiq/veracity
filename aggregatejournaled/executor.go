package aggregate

import (
	"context"
	"errors"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/parcel"
	"golang.org/x/exp/slices"
)

// ErrConflict is an error that indicates an OCC failure when attempting to
// write to a journal or event stream.
var ErrConflict = errors.New("optimistic concurrency conflict")

type Snapshot struct {
	UnacknowledgedCommandIDs map[string]struct{}
	UnpublishedEvents        []parcel.Parcel
	Root                     dogma.AggregateRoot
}

type commandExecuted struct {
	CommandID string
	Events    []parcel.Parcel
}

func (r commandExecuted) ApplyTo(s *Snapshot) {
	if s.UnacknowledgedCommandIDs == nil {
		s.UnacknowledgedCommandIDs = map[string]struct{}{}
	}

	s.UnacknowledgedCommandIDs[r.CommandID] = struct{}{}
	s.UnpublishedEvents = r.Events
}

type commandAcknowledged struct {
	CommandID string
}

func (r commandAcknowledged) ApplyTo(s *Snapshot) {
	delete(s.UnacknowledgedCommandIDs, r.CommandID)
}

type CommandExecutor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Journal         Journal
	Stream          EventStream
	Packer          *parcel.Packer

	snapshot      Snapshot
	journalOffset uint64
	streamOffset  uint64
}

func (e *CommandExecutor) Load(ctx context.Context) error {
	e.snapshot = Snapshot{
		Root: e.Handler.New(),
	}

	e.journalOffset = 0
	for {
		entries, err := e.Journal.Read(
			ctx,
			e.HandlerIdentity.Key,
			e.InstanceID,
			e.journalOffset,
		)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			break
		}

		for _, entry := range entries {
			e.journalOffset++
			entry.ApplyTo(&e.snapshot)
		}
	}

	e.streamOffset = 0
	for {
		if len(e.snapshot.UnpublishedEvents) == 0 {
			return nil
		}

		events, err := e.Stream.Read(ctx, e.streamOffset)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}

		for _, ev := range events {
			e.streamOffset++

			if i := slices.IndexFunc(
				e.snapshot.UnpublishedEvents,
				func(x parcel.Parcel) bool {
					return x.ID() == ev.ID()
				},
			); i != -1 {
				e.snapshot.UnpublishedEvents = slices.Delete(
					e.snapshot.UnpublishedEvents,
					i,
					i+1,
				)
			}
		}
	}

	for _, ev := range e.snapshot.UnpublishedEvents {
		err := e.Stream.WriteAtOffset(ctx, e.streamOffset, ev)
		if err != nil {
			return err
		}
		e.streamOffset++
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

	if _, ok := e.snapshot.UnacknowledgedCommandIDs[cmd.ID()]; !ok {
		if err := e.handleCommand(ctx, cmd); err != nil {
			return err
		}
	}

	if err := e.ackCommand(ctx, cmd, ack); err != nil {
		return err
	}

	if len(e.snapshot.UnpublishedEvents) == 0 {
		return nil
	}

	for _, ev := range e.snapshot.UnpublishedEvents {
		if err := e.Stream.WriteAtOffset(ctx, e.streamOffset, ev); err != nil {
			return err
		}
		e.streamOffset++
	}

	e.snapshot.UnpublishedEvents = nil

	return nil
}

func (e *CommandExecutor) handleCommand(
	ctx context.Context,
	cmd parcel.Parcel,
) error {
	s := &scope{
		Executor: e,
		Command:  cmd,
	}
	e.Handler.HandleCommand(e.snapshot.Root, s, cmd.Message)

	return e.writeToJournal(
		ctx,
		&commandExecuted{
			CommandID: cmd.ID(),
			Events:    s.Events,
		},
	)
}

func (e *CommandExecutor) ackCommand(
	ctx context.Context,
	cmd parcel.Parcel,
	ack func(context.Context) error,
) error {
	if err := ack(ctx); err != nil {
		return err
	}

	return e.writeToJournal(
		ctx,
		&commandAcknowledged{
			CommandID: cmd.ID(),
		},
	)
}

func (e *CommandExecutor) writeToJournal(ctx context.Context, r JournalEntry) error {
	if err := e.Journal.Write(
		ctx,
		e.HandlerIdentity.Key,
		e.InstanceID,
		e.journalOffset,
		r,
	); err != nil {
		return err
	}

	r.ApplyTo(&e.snapshot)
	e.journalOffset++

	return nil
}
