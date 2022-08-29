package queue

import (
	"container/heap"
	"context"

	"github.com/dogmatiq/veracity/parcel"
)

type Queue struct {
	Journal Journal

	offset   uint64
	messages map[string]*message
	queue    pqueue
}

func (q *Queue) Enqueue(ctx context.Context, p parcel.Parcel) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	if _, ok := q.messages[p.ID()]; ok {
		return nil
	}

	return q.apply(
		ctx,
		EnqueueEntry{p},
	)
}

func (e EnqueueEntry) apply(q *Queue) {
	m := &message{
		Parcel:   e.Parcel,
		Priority: q.offset,
	}

	q.messages[e.Parcel.ID()] = m
	heap.Push(&q.queue, m)
}

func (q *Queue) Acquire(ctx context.Context) (parcel.Parcel, bool, error) {
	if err := q.load(ctx); err != nil {
		return parcel.Parcel{}, false, err
	}

	if q.queue.Len() == 0 {
		return parcel.Parcel{}, false, nil
	}

	p := q.queue.Peek()

	return p, true, q.apply(
		ctx,
		AcquireEntry{p.ID()},
	)
}

func (e AcquireEntry) apply(q *Queue) {
	m := q.messages[e.ID]
	heap.Remove(&q.queue, m.index)
	m.Acquired = true
}

func (q *Queue) Ack(ctx context.Context, id string) error {
	if !q.messages[id].Acquired {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		AckEntry{id},
	)
}

func (e AckEntry) apply(q *Queue) {
	q.messages[e.ID] = nil
}

func (q *Queue) Nack(ctx context.Context, id string) error {
	if !q.messages[id].Acquired {
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
		m := q.messages[id]
		m.Acquired = false
		heap.Push(&q.queue, m)
	}
}

func (q *Queue) load(ctx context.Context) error {
	if q.messages != nil {
		return nil
	}

	q.messages = map[string]*message{}

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

	var acquired []string
	for _, m := range q.messages {
		if m != nil && m.Acquired {
			acquired = append(acquired, m.Parcel.ID())
		}
	}

	return q.apply(
		ctx,
		NackEntry{acquired},
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
