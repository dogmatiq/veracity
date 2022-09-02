package queue

import (
	"container/heap"
	"context"
	"errors"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"go.uber.org/zap"
)

// Queue is a durable, ordered queue of messages.
type Queue struct {
	Journal journal.Journal[*JournalRecord]
	Logger  *zap.Logger

	version  uint64
	messages map[string]*message
	acquired map[string]*message
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
			Enqueue: &EnqueueRecord{
				Envelope: env,
			},
		},
	)
}

func (q *Queue) applyEnqueue(rec *EnqueueRecord) {
	m := &message{
		Envelope: rec.GetEnvelope(),
		Priority: q.version,
	}

	id := m.Envelope.GetMessageId()
	q.messages[id] = m
	heap.Push(&q.queue, m)

	q.Logger.Debug(
		"message added to queue",
		zap.String("id", id),
		zap.String("type", m.Envelope.GetPortableName()),
	)
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
			Acquire: &AcquireRecord{
				MessageId: env.GetMessageId(),
			},
		},
	)
}

func (q *Queue) applyAcquire(rec *AcquireRecord) {
	id := rec.GetMessageId()
	m := q.messages[id]
	q.acquired[id] = m
	heap.Remove(&q.queue, m.index)

	q.Logger.Debug(
		"message acquired from queue",
		zap.String("id", id),
		zap.String("type", m.Envelope.GetPortableName()),
	)
}

// Ack acknowledges a previously acquired message, permanently removing it from
// the queue.
func (q *Queue) Ack(ctx context.Context, id string) error {
	if _, ok := q.acquired[id]; !ok {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		&JournalRecord_Ack{
			Ack: &AckRecord{
				MessageId: id,
			},
		},
	)
}

func (q *Queue) applyAck(rec *AckRecord) {
	id := rec.GetMessageId()
	q.messages[id] = nil
	delete(q.acquired, id)

	q.Logger.Debug(
		"message removed from queue (ack)",
		zap.String("id", id),
	)
}

// Nack negatively acknowledges a previously acquired message, returning it to
// the queue so that it may be re-acquired.
func (q *Queue) Nack(ctx context.Context, id string) error {
	if _, ok := q.acquired[id]; !ok {
		panic("message has not been acquired")
	}

	return q.apply(
		ctx,
		&JournalRecord_Nack{
			Nack: &NackRecord{
				MessageId: id,
			},
		},
	)
}

func (q *Queue) applyNack(rec *NackRecord) {
	id := rec.GetMessageId()
	m := q.acquired[id]
	delete(q.acquired, id)
	heap.Push(&q.queue, m)

	q.Logger.Debug(
		"message returned to queue (nack)",
		zap.String("id", id),
	)
}

// load reads all entries from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.messages != nil {
		return nil
	}

	q.messages = map[string]*message{}
	q.acquired = map[string]*message{}

	for {
		rec, ok, err := q.Journal.Read(ctx, q.version)
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		rec.GetOneOf().(journalRecord).apply(q)
		q.version++
	}

	for id := range q.acquired {
		if err := q.Nack(ctx, id); err != nil {
			return err
		}
	}

	q.Logger.Debug(
		"loaded queue from journal",
		zap.Int("size", q.queue.Len()),
	)

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (q *Queue) apply(
	ctx context.Context,
	rec journalRecord,
) error {
	ok, err := q.Journal.Write(
		ctx,
		q.version,
		&JournalRecord{
			OneOf: rec,
		},
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("optimistic concurrency conflict")
	}

	rec.apply(q)
	q.version++

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