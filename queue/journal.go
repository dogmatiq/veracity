package queue

import (
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/parcel"
)

// Journal is an append-only log used by a queue to persist its state.
type Journal interface {
	journal.Journal[JournalEntry]
}

// JournalEntry is an entry in a queue's journal.
type JournalEntry interface {
	apply(entryVisitor)
}

type (
	// Enqueue is a journal entry that records the enqueuing of some messages.
	Enqueue struct{ Parcels []parcel.Parcel }

	// Acquire is a journal entry that records the acquisition of some messages.
	Acquire struct{ MessageIDs []string }

	// Ack is a journal entry that records the acknowledgement of some messages.
	Ack struct{ MessageIDs []string }

	// Nack is a journal entry that records the negative acknowledgement of some
	// messages.
	Nack struct{ MessageIDs []string }
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
