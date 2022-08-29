package queue

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/parcel"
)

// ErrConflict indicates that a journal entry could not be applied because it
// conflicts with an existing entry at the same offset.
var ErrConflict = errors.New("optimistic concurrency conflict")

// Journal is an append-only log used by a queue to persist its state.
type Journal interface {
	Read(
		ctx context.Context,
		offset uint64,
	) ([]JournalEntry, uint64, error)

	Write(
		ctx context.Context,
		offset uint64,
		e JournalEntry,
	) error
}

// JournalEntry is an entry in a queue's journal.
type JournalEntry interface {
	apply(*Queue)
}

// Enqueue is a journal entry that records the enqueuing of some messages.
type Enqueue struct {
	Parcels []parcel.Parcel
}

// Acquire is a journal entry that records the acquisition of some messages.
type Acquire struct {
	MessageIDs []string
}

// Ack is a journal entry that records the acknowledgement of some messages.
type Ack struct {
	MessageIDs []string
}

// Nack is a journal entry that records the negative acknowledgement of some
// messages.
type Nack struct {
	MessageIDs []string
}
