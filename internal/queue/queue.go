package queue

import (
	"container/heap"
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/journal"
)

// Queue is a durable, ordered queue of messages.
type Queue struct {
	Journal journal.Journal[*JournalRecord]

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
		&JournalRecord_Enqueue{
			Enqueue: env,
		},
	)
}

func (q *Queue) applyEnqueue(env *envelopespec.Envelope) {
	m := &message{
		Envelope: env,
		Priority: q.offset,
	}

	q.messages[env.GetMessageId()] = m
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
		&JournalRecord_Acquire{
			Acquire: env.GetMessageId(),
		},
	)
}

func (q *Queue) applyAcquire(id string) {
	m := q.messages[id]
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
		&JournalRecord_Ack{
			Ack: id,
		},
	)
}

func (q *Queue) applyAck(id string) {
	q.messages[id] = nil
}

// Nack negatively acknowledges a previously acquired message, returning it to
// the queue so that it may be re-acquired.
func (q *Queue) Nack(ctx context.Context, id string) error {
	if !q.messages[id].Acquired {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		&JournalRecord_Nack{
			Nack: id,
		},
	)
}

func (q *Queue) applyNack(id string) {
	m := q.messages[id]
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
		rec, ok, err := q.Journal.Read(ctx, q.offset)
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		rec.GetOneOf().(journalRecord).apply(q)
		q.offset++
	}

	for id, m := range q.messages {
		if m != nil && m.Acquired {
			if err := q.apply(
				ctx,
				&JournalRecord_Nack{
					Nack: id,
				},
			); err != nil {
				return err
			}
		}
	}

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (q *Queue) apply(
	ctx context.Context,
	rec journalRecord,
) error {
	if err := q.Journal.Write(
		ctx,
		q.offset,
		&JournalRecord{
			OneOf: rec,
		},
	); err != nil {
		return err
	}

	rec.apply(q)
	q.offset++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(q *Queue)
}

func (x *JournalRecord_Enqueue) apply(q *Queue) { q.applyEnqueue(x.Enqueue) }
func (x *JournalRecord_Acquire) apply(q *Queue) { q.applyAcquire(x.Acquire) }
func (x *JournalRecord_Ack) apply(q *Queue)     { q.applyAck(x.Ack) }
func (x *JournalRecord_Nack) apply(q *Queue)    { q.applyNack(x.Nack) }
