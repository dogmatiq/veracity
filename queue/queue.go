package queue

import (
	"container/heap"
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// Queue is a durable, ordered queue of messages.
type Queue struct {
	Journal Journal

	offset   uint64
	messages map[string]*message
	queue    pqueue
}

// Enqueue adds a message to the queue.
func (q *Queue) Enqueue(ctx context.Context, env *envelopespec.Envelope) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	if _, ok := q.messages[env.GetMessageId()]; ok {
		return nil
	}

	return q.apply(
		ctx,
		Enqueue{
			Envelope: env,
		},
	)
}

func (q *Queue) applyEnqueue(e Enqueue) {
	m := &message{
		Envelope: e.Envelope,
		Priority: q.offset,
	}

	q.messages[e.Envelope.GetMessageId()] = m
	heap.Push(&q.queue, m)
}

// Acquire acquires a message from the queue for processing.
//
// If the queue is empty ok is false; otherwise, p is the next unacquired
// message in the queue.
//
// The message must be subsequently removed from the queue or returned to the
// pool of unacquired messages by calling Ack() or Nack(), respectively.
func (q *Queue) Acquire(ctx context.Context) (env *envelopespec.Envelope, ok bool, err error) {
	if err := q.load(ctx); err != nil {
		return nil, false, err
	}

	if q.queue.Len() == 0 {
		return nil, false, nil
	}

	env = q.queue.Peek()

	return env, true, q.apply(
		ctx,
		Acquire{
			MessageID: env.GetMessageId(),
		},
	)
}

func (q *Queue) applyAcquire(e Acquire) {
	m := q.messages[e.MessageID]
	heap.Remove(&q.queue, m.index)
	m.Acquired = true
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
			MessageID: id,
		},
	)
}

func (q *Queue) applyAck(e Ack) {
	q.messages[e.MessageID] = nil
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
			MessageID: id,
		},
	)
}

func (q *Queue) applyNack(e Nack) {
	m := q.messages[e.MessageID]
	m.Acquired = false
	heap.Push(&q.queue, m)
}

// load reads all entries from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.messages != nil {
		return nil
	}

	q.messages = map[string]*message{}

	for {
		entries, next, err := q.Journal.Read(ctx, q.offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			break
		}

		for _, e := range entries {
			e.apply(q)
		}

		q.offset = next
	}

	for id, m := range q.messages {
		if m != nil && m.Acquired {
			if err := q.apply(
				ctx,
				Nack{
					MessageID: id,
				},
			); err != nil {
				return err
			}
		}
	}

	return nil
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
