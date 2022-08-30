package executor

import (
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/parcel"
)

// Journal is an append-only log used by an aggregate instance to persist its
// state.
type Journal interface {
	journal.Journal[JournalEntry]
}

// JournalEntry is an entry in an aggregate instance's journal.
type JournalEntry interface {
	apply(entryVisitor)
}

type (
	// XXX is a journal entry that records the enqueuing of some messages.
	XXX struct{ Parcels []parcel.Parcel }
)

// entryVisitor dispatches based on the type of a journal entry.
type entryVisitor interface {
	applyEnqueue(XXX)
}

func (e XXX) apply(v entryVisitor) { v.applyEnqueue(e) }
