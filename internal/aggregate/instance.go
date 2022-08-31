package aggregate

import (
	"context"
	"errors"

	"github.com/dogmatiq/dogma"
	envelopespec "github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/journal"
)

// Acknowledger is a function that acknowledges a queued message.
type Acknowledger interface {
	Ack(ctx context.Context, id string) error
}

type EventWriter interface {
	Write(ctx context.Context, env *envelopespec.Envelope) error
}

// Instance is the in-memory representation of an aggregate instance.
type Instance struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Packer          *envelope.Packer
	Journal         journal.Journal[*JournalRecord]
	EventWriter     EventWriter
	Acknowledger    Acknowledger
	Root            dogma.AggregateRoot

	version uint64
}

func (i *Instance) ExecuteCommand(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	cmd, err := i.Packer.Unpack(env)
	if err != nil {
		return err
	}

	s := &scope{
		ID:   i.InstanceID,
		Root: i.Root,
	}
	i.Handler.HandleCommand(i.Root, s, cmd)

	var envelopes []*envelopespec.Envelope
	for _, ev := range s.Events {
		envelopes = append(
			envelopes,
			i.Packer.Pack(
				ev,
				envelope.WithCause(env),
				envelope.WithHandler(i.HandlerIdentity),
				envelope.WithInstanceID(i.InstanceID),
			),
		)
	}

	if err := i.apply(
		ctx,
		&JournalRecord_Revision{
			Revision: &RevisionRecord{
				CommandId:      env.GetMessageId(),
				EventEnvelopes: envelopes,
			},
		},
	); err != nil {
		return err
	}

	// if err := i.EventWriter.Write(ctx, env); err != nil {
	// 	return err
	// }

	return i.Acknowledger.Ack(ctx, env.GetMessageId())
}

func (i *Instance) applyRevision(rec *RevisionRecord) {
}

// apply writes a record to the journal and applies it to the queue.
func (i *Instance) apply(
	ctx context.Context,
	rec journalRecord,
) error {
	ok, err := i.Journal.Write(
		ctx,
		i.version,
		&JournalRecord{
			OneOf: rec,
		},
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("optimistic concurrency conflict")
	}

	rec.apply(i)
	i.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(i *Instance)
}

func (x *JournalRecord_Revision) apply(i *Instance) { i.applyRevision(x.Revision) }

// func (x *JournalRecord_Acquire) apply(i *Instance) { q.applyAcquire(x.Acquire) }
// func (x *JournalRecord_Ack) apply(i *Instance)     { q.applyAck(x.Ack) }
// func (x *JournalRecord_Nack) apply(i *Instance)    { q.applyNack(x.Nack) }
