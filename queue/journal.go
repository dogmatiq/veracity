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
	ApplyTo(*Queue)
}

type EnqueueEntry struct {
	Parcel parcel.Parcel
}

func (e EnqueueEntry) ApplyTo(q *Queue) {
	q.pending[e.Parcel.ID()] = e.Parcel
}

type AcquireEntry struct {
	ID string
}

func (e AcquireEntry) ApplyTo(q *Queue) {
	q.unacknowledged[e.ID] = q.pending[e.ID]
	delete(q.pending, e.ID)
}

type AckEntry struct {
	ID string
}

func (e AckEntry) ApplyTo(q *Queue) {
	q.acknowledged[e.ID] = struct{}{}
	delete(q.unacknowledged, e.ID)
}

type NackEntry struct {
	IDs []string
}

func (e NackEntry) ApplyTo(q *Queue) {
	for _, id := range e.IDs {
		q.pending[id] = q.unacknowledged[id]
		delete(q.unacknowledged, id)
	}
}
