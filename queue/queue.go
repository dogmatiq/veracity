package queue

import (
	"context"

	"github.com/dogmatiq/veracity/parcel"
	"golang.org/x/exp/maps"
)

type Queue struct {
	Journal Journal

	offset         uint64
	pending        map[string]parcel.Parcel
	unacknowledged map[string]parcel.Parcel
	acknowledged   map[string]struct{}
	order          []string
}

func (q *Queue) Enqueue(ctx context.Context, m parcel.Parcel) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	if _, ok := q.acknowledged[m.ID()]; ok {
		return nil
	}

	return q.apply(
		ctx,
		EnqueueEntry{m},
	)
}

func (e EnqueueEntry) apply(q *Queue) {
	q.pending[e.Parcel.ID()] = e.Parcel
	q.order = append(q.order, e.Parcel.ID())
}

func (q *Queue) Acquire(ctx context.Context) (parcel.Parcel, bool, error) {
	if err := q.load(ctx); err != nil {
		return parcel.Parcel{}, false, err
	}

	for _, id := range q.order {
		if m, ok := q.pending[id]; ok {
			return m, true, q.apply(
				ctx,
				AcquireEntry{id},
			)
		}
	}

	return parcel.Parcel{}, false, nil
}

func (e AcquireEntry) apply(q *Queue) {
	q.unacknowledged[e.ID] = q.pending[e.ID]
	delete(q.pending, e.ID)
}

func (q *Queue) Ack(ctx context.Context, id string) error {
	if _, ok := q.unacknowledged[id]; !ok {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		AckEntry{id},
	)
}

func (e AckEntry) apply(q *Queue) {
	q.acknowledged[e.ID] = struct{}{}
	delete(q.unacknowledged, e.ID)
}

func (q *Queue) Nack(ctx context.Context, id string) error {
	if _, ok := q.unacknowledged[id]; !ok {
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
		q.pending[id] = q.unacknowledged[id]
		delete(q.unacknowledged, id)
	}
}

func (q *Queue) load(ctx context.Context) error {
	if q.pending != nil {
		return nil
	}

	q.pending = map[string]parcel.Parcel{}
	q.unacknowledged = map[string]parcel.Parcel{}
	q.acknowledged = map[string]struct{}{}

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

	if len(q.unacknowledged) == 0 {
		return nil
	}

	return q.apply(
		ctx,
		NackEntry{
			maps.Keys(q.unacknowledged),
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
