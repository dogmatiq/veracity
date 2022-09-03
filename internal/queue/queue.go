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

// Enqueue adds messages to the queue.
func (q *Queue) Enqueue(ctx context.Context, envelopes ...*envelopespec.Envelope) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	rec := &JournalRecord_Enqueue{
		Enqueue: &EnqueueRecord{},
	}

	for _, env := range envelopes {
		if _, ok := q.messages[env.GetMessageId()]; ok {
			q.Logger.Debug(
				"message ignored because it is already enqueued",
				zap.Uint64("queue_version", q.version),
				zap.Int("queue_size", q.len()),
				zap.String("message_id", env.GetMessageId()),
				zap.String("message_type", env.GetPortableName()),
				zap.String("message_desc", env.GetDescription()),
			)
			continue
		}

		rec.Enqueue.Envelopes = append(rec.Enqueue.Envelopes, env)
	}

	if len(rec.Enqueue.Envelopes) == 0 {
		return nil
	}

	if err := q.apply(ctx, rec); err != nil {
		return err
	}

	for _, env := range rec.Enqueue.GetEnvelopes() {
		q.Logger.Debug(
			"message enqueued",
			zap.Uint64("queue_version", q.version),
			zap.Int("queue_size", q.len()),
			zap.String("message_id", env.GetMessageId()),
			zap.String("message_type", env.GetPortableName()),
			zap.String("message_desc", env.GetDescription()),
		)
	}

	return nil
}

func (q *Queue) applyEnqueue(rec *EnqueueRecord) {
	for _, env := range rec.GetEnvelopes() {
		m := &message{
			Envelope: env,
			Priority: q.version,
		}

		id := m.Envelope.GetMessageId()
		q.messages[id] = m
		heap.Push(&q.queue, m)
	}
}

// Acquire acquires a message from the queue for processing.
//
// If the queue is empty ok is false; otherwise, env is the next unacquired
// message in the queue.
//
// The message must be subsequently removed from the queue or returned to the
// pool of unacquired messages by calling Ack() or Reject(), respectively.
func (q *Queue) Acquire(ctx context.Context) (env *envelopespec.Envelope, ok bool, err error) {
	if err := q.load(ctx); err != nil {
		return nil, false, err
	}

	if q.queue.Len() == 0 {
		return nil, false, nil
	}

	env = q.queue.Peek()

	if err := q.apply(
		ctx,
		&JournalRecord_Acquire{
			Acquire: &AcquireRecord{
				MessageId: env.GetMessageId(),
			},
		},
	); err != nil {
		return nil, false, err
	}

	q.Logger.Debug(
		"message acquired",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.len()),
		zap.String("message_id", env.GetMessageId()),
		zap.String("message_type", env.GetPortableName()),
		zap.String("message_desc", env.GetDescription()),
	)

	return env, true, nil
}

func (q *Queue) applyAcquire(rec *AcquireRecord) {
	id := rec.GetMessageId()
	m := q.messages[id]
	q.acquired[id] = m
	heap.Remove(&q.queue, m.index)
}

// Ack acknowledges a previously acquired message, permanently removing it from
// the queue.
func (q *Queue) Ack(ctx context.Context, id string) error {
	m, ok := q.acquired[id]
	if !ok {
		panic("message has not been acquired")
	}

	if err := q.apply(
		ctx,
		&JournalRecord_Ack{
			Ack: &AckRecord{
				MessageId: id,
			},
		},
	); err != nil {
		return err
	}

	q.Logger.Debug(
		"message acknowledged",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.len()),
		zap.String("message_id", m.Envelope.GetMessageId()),
		zap.String("message_type", m.Envelope.GetPortableName()),
		zap.String("message_desc", m.Envelope.GetDescription()),
	)

	return nil
}

func (q *Queue) applyAck(rec *AckRecord) {
	id := rec.GetMessageId()
	q.messages[id] = nil
	delete(q.acquired, id)
}

// Reject returns previously acquired message to the queue so that it may be
// re-acquired.
func (q *Queue) Reject(ctx context.Context, id string) error {
	m, ok := q.acquired[id]
	if !ok {
		panic("message has not been acquired")
	}

	if err := q.apply(
		ctx,
		&JournalRecord_Reject{
			Reject: &RejectRecord{
				MessageId: id,
			},
		},
	); err != nil {
		return err
	}

	q.Logger.Debug(
		"message rejected",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.len()),
		zap.String("message_id", id),
		zap.String("message_type", m.Envelope.GetPortableName()),
		zap.String("message_desc", m.Envelope.GetDescription()),
	)

	return nil
}

func (q *Queue) applyReject(rec *RejectRecord) {
	id := rec.GetMessageId()
	m := q.acquired[id]
	delete(q.acquired, id)
	heap.Push(&q.queue, m)
}

// len returns the number of messages on the queue.
func (q *Queue) len() int {
	return q.queue.Len() + len(q.acquired)
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

		q.version++
		rec.GetOneOf().(journalRecord).apply(q)
	}

	n := len(q.acquired)
	for id := range q.acquired {
		if err := q.Reject(ctx, id); err != nil {
			return err
		}
	}

	q.Logger.Debug(
		"loaded queue from journal",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.len()),
		zap.Int("requeue_count", n),
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
func (x *JournalRecord_Reject) apply(q *Queue)  { q.applyReject(x.Reject) }
