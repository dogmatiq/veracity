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

func (q *Queue) Acquire(ctx context.Context) (parcel.Parcel, bool, error) {
	if err := q.load(ctx); err != nil {
		return parcel.Parcel{}, false, err
	}

	for _, m := range q.pending {
		return m, true, q.apply(
			ctx,
			AcquireEntry{
				m.ID(),
			},
		)
	}

	return parcel.Parcel{}, false, nil
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
			e.ApplyTo(q)
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

	e.ApplyTo(q)
	q.offset++

	return nil
}
