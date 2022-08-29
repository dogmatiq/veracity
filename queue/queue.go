package queue

import (
	"container/heap"
	"context"

	"github.com/dogmatiq/veracity/parcel"
)

// Queue is a durable, ordered queue of messages.
type Queue struct {
	Journal Journal

	offset   uint64
	messages map[string]*message
	queue    pqueue
}

// Enqueue adds a message to the queue.
func (q *Queue) Enqueue(ctx context.Context, p parcel.Parcel) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	if _, ok := q.messages[p.ID()]; ok {
		return nil
	}

	return q.apply(
		ctx,
		Enqueue{
			Parcels: []parcel.Parcel{p},
		},
	)
}

func (e Enqueue) apply(q *Queue) {
	for _, p := range e.Parcels {
		m := &message{
			Parcel:   p,
			Priority: q.offset,
		}

		q.messages[p.ID()] = m
		heap.Push(&q.queue, m)
	}
}

// Acquire acquires a message from the queue for processing.
//
// If the queue is empty ok is false; otherwise, p is the next unacquired
// message in the queue.
//
// The message must be subsequently removed from the queue or returned to the
// pool of unacquired messages by calling Ack() or Nack(), respectively.
func (q *Queue) Acquire(ctx context.Context) (p parcel.Parcel, ok bool, err error) {
	if err := q.load(ctx); err != nil {
		return parcel.Parcel{}, false, err
	}

	if q.queue.Len() == 0 {
		return parcel.Parcel{}, false, nil
	}

	p = q.queue.Peek()

	return p, true, q.apply(
		ctx,
		Acquire{
			MessageIDs: []string{
				p.ID(),
			},
		},
	)
}

func (e Acquire) apply(q *Queue) {
	for _, id := range e.MessageIDs {
		m := q.messages[id]
		heap.Remove(&q.queue, m.index)
		m.Acquired = true
	}
}

// Ack acknowledges a previously acquired message, permanently removing it from
// the queue.
func (q *Queue) Ack(ctx context.Context, id string) error {
	if !q.messages[id].Acquired {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		Ack{
			MessageIDs: []string{id},
		},
	)
}

func (e Ack) apply(q *Queue) {
	for _, id := range e.MessageIDs {
		q.messages[id] = nil
	}
}

// Nack negatively acknowledges a previously acquired message, returning it to
// the queue so that it may be re-acquired.
func (q *Queue) Nack(ctx context.Context, id string) error {
	if !q.messages[id].Acquired {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		Nack{
			MessageIDs: []string{id},
		},
	)
}

func (e Nack) apply(q *Queue) {
	for _, id := range e.MessageIDs {
		m := q.messages[id]
		m.Acquired = false
		heap.Push(&q.queue, m)
	}
}

// load reads all entries from the journal and applies them to the queue.
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
		Nack{
			MessageIDs: acquired,
		},
	)
}

// apply writes an entry to the journal and applies it to the queue.
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
