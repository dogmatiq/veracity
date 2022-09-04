package queue

import (
	"container/heap"
	"context"
	"errors"
	"fmt"

	envelopespec "github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"github.com/dogmatiq/veracity/internal/zapx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Queue is a durable priority queue of messages.
type Queue struct {
	Journal journal.Journal[*JournalRecord]
	Logger  *zap.Logger
	Key     func(m Message) string

	version  uint32
	elements map[string]*elem
	size     int
	queue    pqueue
}

// Message is a container for a message on a queue.
type Message struct {
	Envelope *envelopespec.Envelope
}

// AcquiredMessage is a message that was acquired from a queue.
type AcquiredMessage struct {
	Envelope *envelopespec.Envelope

	queue   *Queue
	element *elem
}

// Enqueue adds messages to the queue.
func (q *Queue) Enqueue(
	ctx context.Context,
	messages ...Message,
) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	makeKey := q.Key
	if makeKey == nil {
		makeKey = func(m Message) string {
			return m.Envelope.MessageId
		}
	}

	r := &JournalRecord_Enqueue{
		Enqueue: &EnqueueRecord{
			Messages: make([]*JournalMessage, 0, len(messages)),
		},
	}

	for _, m := range messages {
		if err := m.Envelope.Validate(); err != nil {
			panic(err)
		}

		key := makeKey(m)

		if _, ok := q.elements[key]; ok {
			// q.Logger.Debug(
			// 	"message ignored because it is already enqueued",
			// 	zap.String("key", key),
			// 	zapx.Envelope("message", m.Envelope),
			// 	zap.Object("queue", (*logAdaptor)(q)),
			// )
		} else {
			t, err := marshalkit.UnmarshalEnvelopeTime(m.Envelope.CreatedAt)
			if err != nil {
				panic(err)
			}

			r.Enqueue.Messages = append(
				r.Enqueue.Messages,
				&JournalMessage{
					Envelope: m.Envelope,
					Key:      key,
					Priority: t.UnixNano(),
				},
			)
		}
	}

	if len(r.Enqueue.Messages) == 0 {
		return nil
	}

	if err := q.apply(ctx, r); err != nil {
		return fmt.Errorf("unable to enqueue messages: %w", err)
	}

	for _, m := range r.Enqueue.Messages {
		q.Logger.Debug(
			"message enqueued",
			zap.String("key", m.Key),
			zapx.Envelope("message", m.Envelope),
			zap.Object("queue", (*logAdaptor)(q)),
		)
	}

	return nil
}

func (q *Queue) applyEnqueue(r *EnqueueRecord) {
	for _, jm := range r.GetMessages() {
		e := &elem{JournalMessage: jm}
		q.elements[jm.Key] = e
		heap.Push(&q.queue, e)
		q.size++
	}
}

// Acquire acquires a message from the queue for processing.
//
// If the queue is empty ok is false; otherwise, m is the next unacquired
// message in the queue.
//
// The message must be subsequently removed from the queue or returned to the
// pool of unacquired messages by calling Ack() or Reject(), respectively.
func (q *Queue) Acquire(ctx context.Context) (m AcquiredMessage, ok bool, err error) {
	if err := q.load(ctx); err != nil {
		return AcquiredMessage{}, false, err
	}

	if q.queue.Len() == 0 {
		return AcquiredMessage{}, false, nil
	}

	e := q.queue.elements[0]
	r := &JournalRecord_Acquire{
		Acquire: &AcquireRecord{
			Key: e.Key,
		},
	}

	if err := q.apply(ctx, r); err != nil {
		return AcquiredMessage{}, false, fmt.Errorf("unable to acquire message: %w", err)
	}

	q.Logger.Debug(
		"message acquired",
		zap.String("key", e.Key),
		zapx.Envelope("message", e.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return AcquiredMessage{
		Envelope: e.Envelope,
		queue:    q,
		element:  e,
	}, true, nil
}

func (q *Queue) applyAcquire(r *AcquireRecord) {
	e := q.elements[r.Key]
	e.acquired = true
	heap.Remove(&q.queue, e.index)
}

// Ack acknowledges a previously acquired message, permanently removing it from
// the queue.
func (q *Queue) Ack(ctx context.Context, m AcquiredMessage) error {
	if m.queue != q || m.element == nil || !m.element.acquired {
		panic("message has not been acquired from this queue")
	}

	r := &JournalRecord_Ack{
		Ack: &AckRecord{
			Key: m.element.Key,
		},
	}

	if err := q.apply(ctx, r); err != nil {
		return fmt.Errorf("unable to acknowledge message: %w", err)
	}

	q.Logger.Debug(
		"message acknowledged",
		zap.String("key", m.element.Key),
		zapx.Envelope("message", m.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return nil
}

func (q *Queue) applyAck(r *AckRecord) {
	q.elements[r.Key] = nil
	q.size--
}

// Reject returns previously acquired message to the queue so that it may be
// re-acquired.
func (q *Queue) Reject(ctx context.Context, m AcquiredMessage) error {
	if m.queue != q || m.element == nil || !m.element.acquired {
		panic("message has not been acquired from this queue")
	}

	r := &JournalRecord_Reject{
		Reject: &RejectRecord{
			Key: m.element.Key,
		},
	}

	if err := q.apply(ctx, r); err != nil {
		return fmt.Errorf("unable to reject message: %w", err)
	}

	q.Logger.Debug(
		"message rejected",
		zap.String("key", m.element.Key),
		zapx.Envelope("message", m.element.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return nil
}

func (q *Queue) applyReject(r *RejectRecord) {
	e := q.elements[r.Key]
	e.acquired = false
	heap.Push(&q.queue, e)
}

// load reads all entries from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.elements != nil {
		return nil
	}

	q.elements = map[string]*elem{}

	for {
		r, ok, err := q.Journal.Read(ctx, q.version)
		if err != nil {
			return fmt.Errorf("unable to load queue: %w", err)
		}
		if !ok {
			break
		}

		r.GetOneOf().(journalRecord).apply(q)
		q.version++
	}

	unacknowledged := 0
	for _, e := range q.elements {
		if e == nil || !e.acquired {
			continue
		}

		r := &JournalRecord_Reject{
			Reject: &RejectRecord{
				Key: e.Key,
			},
		}

		if err := q.apply(ctx, r); err != nil {
			return fmt.Errorf("unable to load queue: %w", err)
		}

		unacknowledged++
	}

	q.Logger.Debug(
		"loaded queue from journal",
		zap.Object("queue", (*logAdaptor)(q)),
		zap.Int("unacknowledged_count", unacknowledged),
	)

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (q *Queue) apply(
	ctx context.Context,
	r journalRecord,
) error {
	ok, err := q.Journal.Write(
		ctx,
		q.version,
		&JournalRecord{
			OneOf: r,
		},
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("optimistic concurrency conflict")
	}

	r.apply(q)
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

type logAdaptor Queue

func (a *logAdaptor) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint32("version", a.version)
	enc.AddInt("size", a.size)
	return nil
}
