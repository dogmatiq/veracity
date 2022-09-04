package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/zapx"
	"go.uber.org/zap"
)

type InstanceSupervisor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Root            dogma.AggregateRoot
	EventAppender   EventAppender
	Packer          *envelope.Packer
	Logger          *zap.Logger
}

func (s *InstanceSupervisor) ExecuteCommand(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	cmd, err := s.Packer.Unpack(env)
	if err != nil {
		return err
	}

	r := &RevisionRecord{}

	s.Handler.HandleCommand(
		s.Root,
		&scope{
			HandlerIdentity: s.HandlerIdentity,
			InstID:          s.InstanceID,
			Root:            s.Root,
			Packer:          s.Packer,
			Logger:          s.Logger.With(zapx.Envelope("command", env)),
			CommandEnvelope: env,
			Revision:        r,
		},
		cmd,
	)

	var envelopes []*envelopespec.Envelope
	for _, a := range r.Actions {
		if r := a.GetRecordEvent(); r != nil {
			envelopes = append(envelopes, r.Envelope)
		}
	}

	if len(envelopes) == 0 {
		return nil
	}

	return s.EventAppender.Append(ctx, envelopes...)
}

func (s *InstanceSupervisor) load(ctx context.Context) error {
	if s.Root != nil {
		return nil
	}

	s.Root = s.Handler.New()
	return nil
}

// func (s *InstanceSupervisor) ExecuteCommand(
// 	ctx context.Context,
// 	env *envelopespec.Envelope,
// ) error {
// 	return nil
// }

// // instance is the in-memory representation of an aggregate instance.
// type instance struct {
// 	// Executor *executor

// 	// HandlerIdentity *envelopespec.Identity
// 	// InstanceID      string
// 	// Handler         dogma.AggregateMessageHandler
// 	// Packer          *envelope.Packer
// 	// EventAppender   EventAppender
// 	// Journal         journal.Journal[*JournalRecord]
// 	// Logger          *zap.Logger
// 	// Root            dogma.AggregateRoot

// 	// version uint32
// 	// commands map[string]struct{}
// 	// events   []*envelopespec.Envelope
// }

// // func (i *Instance) ExecuteCommand(
// // 	ctx context.Context,
// // 	env *envelopespec.Envelope,
// // ) error {
// // 	if err := i.load(ctx); err != nil {
// // 		return err
// // 	}

// // 	if len(i.events) != 0 {
// // 		panic("cannot handle a command while there are unpublished events")
// // 	}

// // 	if _, ok := i.commands[env.GetMessageId()]; ok {
// // 		return nil
// // 	}

// // 	cmd, err := i.Packer.Unpack(env)
// // 	if err != nil {
// // 		return err
// // 	}

// // 	s := &scope{
// // 		InstID:          i,
// // 		CommandEnvelope: env,
// // 	}
// // 	i.Handler.HandleCommand(i.Root, s, cmd)

// // 	if err := i.apply(
// // 		ctx,
// // 		&JournalRecord_Revision{
// // 			Revision: &RevisionRecord{
// // 				CommandId:      env.GetMessageId(),
// // 				EventEnvelopes: s.EventEnvelopes,
// // 			},
// // 		},
// // 	); err != nil {
// // 		return err
// // 	}

// // 	for _, env := range i.events {
// // 		if err := i.EventAppender.Append(ctx, env); err != nil {
// // 			return err
// // 		}
// // 	}

// // 	return nil
// // }

// // func (i *Instance) applyRevision(rec *RevisionRecord) {
// // 	i.commands[rec.GetCommandId()] = struct{}{}
// // 	i.events = rec.GetEventEnvelopes()
// // }

// // func (i *Instance) applyEventsWritten(rec *EventsWrittenRecord) {
// // 	i.events = nil
// // }

// // // load reads all entries from the journal and applies them to the queue.
// // func (i *Instance) load(ctx context.Context) error {
// // 	if i.commands != nil {
// // 		return nil
// // 	}

// // 	i.commands = map[string]struct{}{}

// // 	for {
// // 		rec, ok, err := i.Journal.Read(ctx, i.version)
// // 		if err != nil {
// // 			return err
// // 		}
// // 		if !ok {
// // 			break
// // 		}

// // 		rec.GetOneOf().(journalRecord).apply(i)
// // 		i.version++
// // 	}

// // 	n := len(i.events)
// // 	if n != 0 {
// // 		for _, env := range i.events {
// // 			if err := i.EventAppender.Append(ctx, env); err != nil {
// // 				return err
// // 			}
// // 		}

// // 		if err := i.apply(
// // 			ctx,
// // 			&JournalRecord_EventsWritten{
// // 				EventsWritten: &EventsWrittenRecord{},
// // 			},
// // 		); err != nil {
// // 			return err
// // 		}
// // 	}

// // 	i.Logger.Debug(
// // 		"loaded aggregate instance from journal",
// // 		zap.Uint32("version", i.version),
// // 		zap.Int("unpublished_events", n),
// // 	)

// // 	return nil
// // }

// // // apply writes a record to the journal and applies it to the queue.
// // func (i *Instance) apply(
// // 	ctx context.Context,
// // 	rec journalRecord,
// // ) error {
// // 	ok, err := i.Journal.Write(
// // 		ctx,
// // 		i.version,
// // 		&JournalRecord{
// // 			OneOf: rec,
// // 		},
// // 	)
// // 	if err != nil {
// // 		return err
// // 	}
// // 	if !ok {
// // 		return errors.New("optimistic concurrency conflict")
// // 	}

// // 	rec.apply(i)
// // 	i.version++

// // 	return nil
// // }

// // type journalRecord interface {
// // 	isJournalRecord_OneOf
// // 	apply(i *Instance)
// // }

// // func (x *JournalRecord_Revision) apply(i *Instance)      { i.applyRevision(x.Revision) }
// // func (x *JournalRecord_EventsWritten) apply(i *Instance) { i.applyEventsWritten(x.EventsWritten) }
