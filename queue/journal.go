package queue

import (
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/persistence/occjournal"
)

// Journal is an append-only log used by a queue to persist its state.
type Journal interface {
	occjournal.Journal[JournalEntry]
}

// JournalEntry is an entry in a queue's journal.
type JournalEntry interface {
	apply(entryVisitor)
}

type (
	// Enqueue is a journal entry that records the enqueuing of some messages.
	Enqueue struct{ Envelope *envelopespec.Envelope }

	// Acquire is a journal entry that records the acquisition of some messages.
	Acquire struct{ MessageID string }

	// Ack is a journal entry that records the acknowledgement of some messages.
	Ack struct{ MessageID string }

	// Nack is a journal entry that records the negative acknowledgement of some
	// messages.
	Nack struct{ MessageID string }
)

// entryVisitor dispatches based on the type of a journal entry.
type entryVisitor interface {
	applyEnqueue(Enqueue)
	applyAcquire(Acquire)
	applyAck(Ack)
	applyNack(Nack)
}

func (e Enqueue) apply(v entryVisitor) { v.applyEnqueue(e) }
func (e Acquire) apply(v entryVisitor) { v.applyAcquire(e) }
func (e Ack) apply(v entryVisitor)     { v.applyAck(e) }
func (e Nack) apply(v entryVisitor)    { v.applyNack(e) }
