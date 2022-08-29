package queue

import (
	"context"

	"github.com/dogmatiq/veracity/parcel"
	"golang.org/x/exp/maps"
)

type Queue struct {
	Journal Journal

	offset      uint64
	deduplicate map[string]struct{}
	pending     map[string]parcel.Parcel
	enqueued    []string
	acquired    map[string]struct{}
}

func (q *Queue) Enqueue(ctx context.Context, m parcel.Parcel) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	if _, ok := q.deduplicate[m.ID()]; ok {
		return nil
	}

	return q.apply(
		ctx,
		EnqueueEntry{m},
	)
}

func (e EnqueueEntry) apply(q *Queue) {
	id := e.Parcel.ID()
	q.deduplicate[id] = struct{}{}
	q.pending[id] = e.Parcel
	q.enqueued = append(q.enqueued, id)
}

func (q *Queue) Acquire(ctx context.Context) (parcel.Parcel, bool, error) {
	if err := q.load(ctx); err != nil {
		return parcel.Parcel{}, false, err
	}

	if len(q.enqueued) == 0 {
		return parcel.Parcel{}, false, nil
	}

	id := q.enqueued[0]
	m := q.pending[id]

	return m, true, q.apply(
		ctx,
		AcquireEntry{id},
	)
}

func (e AcquireEntry) apply(q *Queue) {
	if e.ID != q.enqueued[0] {
		panic("acquired message is not at the head of the queue")
	}

	q.enqueued = q.enqueued[1:]
	q.acquired[e.ID] = struct{}{}
}

func (q *Queue) Ack(ctx context.Context, id string) error {
	if _, ok := q.acquired[id]; !ok {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		AckEntry{id},
	)
}

func (e AckEntry) apply(q *Queue) {
	delete(q.pending, e.ID)
	delete(q.acquired, e.ID)
}

func (q *Queue) Nack(ctx context.Context, id string) error {
	if _, ok := q.acquired[id]; !ok {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		NackEntry{
			[]string{id},
		},
	)
}

func (e NackEntry) apply(q *Queue) {
	for _, id := range e.IDs {
		delete(q.acquired, id)
	}

	q.enqueued = append(q.enqueued, e.IDs...)
}

func (q *Queue) load(ctx context.Context) error {
	if q.deduplicate != nil {
		return nil
	}

	q.deduplicate = map[string]struct{}{}
	q.pending = map[string]parcel.Parcel{}
	q.acquired = map[string]struct{}{}

	for {
		entries, offset, err := q.Journal.Read(ctx, q.offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			break
		}

		for _, e := range entries {
			e.apply(q)
		}

		q.offset = offset
	}

	if len(q.acquired) == 0 {
		return nil
	}

	return q.apply(
		ctx,
		NackEntry{
			maps.Keys(q.acquired),
		},
	)
}

func (q *Queue) apply(
	ctx context.Context,
	e JournalEntry,
) error {
	if err := q.Journal.Write(ctx, q.offset, e); err != nil {
		return err
	}

	e.apply(q)
	q.offset++

	return nil
}
