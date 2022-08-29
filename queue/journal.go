package queue

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/parcel"
)

var ErrConflict = errors.New("optimistic concurrency conflict")

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

type JournalEntry interface {
	apply(*Queue)
}

type EnqueueEntry struct {
	Parcel parcel.Parcel
}

type AcquireEntry struct {
	ID string
}

type AckEntry struct {
	ID string
}

type NackEntry struct {
	IDs []string
}
