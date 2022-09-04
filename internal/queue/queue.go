package queue

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/interopspec/envelopespec"
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

	version  uint32
	messages map[string]*message
	acquired map[string]*message
	queue    pqueue
}

// Enqueue adds messages to the queue.
func (q *Queue) Enqueue(
	ctx context.Context,
	envelopes ...*envelopespec.Envelope,
) error {
	if err := q.load(ctx); err != nil {
		return err
	}

	r := &JournalRecord_Enqueue{
		Enqueue: &EnqueueRecord{},
	}

	for _, env := range envelopes {
		if err := env.Validate(); err != nil {
			panic(err)
		}

		if _, ok := q.messages[env.GetMessageId()]; ok {
			q.Logger.Debug(
				"message ignored because it is already enqueued",
				zap.Object("queue", (*logAdaptor)(q)),
				zapx.Envelope("message", env),
			)
		} else {
			r.Enqueue.Envelopes = append(r.Enqueue.Envelopes, env)
		}
	}

	if len(r.Enqueue.Envelopes) == 0 {
		return nil
	}

	if err := q.apply(
		ctx,
		r,
	); err != nil {
		return fmt.Errorf("unable to enqueue messages: %w", err)
	}

	for _, env := range r.Enqueue.Envelopes {
		q.Logger.Debug(
			"message enqueued",
			zap.Object("queue", (*logAdaptor)(q)),
			zapx.Envelope("message", env),
		)
	}

	return nil
}

func (q *Queue) applyEnqueue(r *EnqueueRecord) {
	for _, env := range r.GetEnvelopes() {
		t, err := marshalkit.UnmarshalEnvelopeTime(env.GetCreatedAt())
		if err != nil {
			panic(err)
		}

		m := &message{
			Envelope:  env,
			CreatedAt: t,
		}

		id := m.Envelope.GetMessageId()
		q.messages[id] = m
		q.queue.PushMessage(m)
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

	m := q.queue.PeekMessage()

	if err := q.apply(
		ctx,
		&JournalRecord_Acquire{
			Acquire: &AcquireRecord{
				MessageId: m.Envelope.GetMessageId(),
			},
		},
	); err != nil {
		return nil, false, fmt.Errorf("unable to acquire message: %w", err)
	}

	q.Logger.Debug(
		"message acquired",
		zap.Object("queue", (*logAdaptor)(q)),
		zapx.Envelope("message", m.Envelope),
	)

	return m.Envelope, true, nil
}

func (q *Queue) applyAcquire(r *AcquireRecord) {
	id := r.GetMessageId()
	m := q.messages[id]
	q.acquired[id] = m
	q.queue.RemoveMessage(m)
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
		return fmt.Errorf("unable to acknowledge message: %w", err)
	}

	q.Logger.Debug(
		"message acknowledged",
		zap.Object("queue", (*logAdaptor)(q)),
		zapx.Envelope("message", m.Envelope),
	)

	return nil
}

func (q *Queue) applyAck(r *AckRecord) {
	id := r.GetMessageId()
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
		return fmt.Errorf("unable to reject message: %w", err)
	}

	q.Logger.Debug(
		"message rejected",
		zap.Object("queue", (*logAdaptor)(q)),
		zapx.Envelope("message", m.Envelope),
	)

	return nil
}

func (q *Queue) applyReject(r *RejectRecord) {
	id := r.GetMessageId()
	m := q.acquired[id]
	delete(q.acquired, id)
	q.queue.PushMessage(m)
}

// load reads all entries from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.messages != nil {
		return nil
	}

	q.messages = map[string]*message{}
	q.acquired = map[string]*message{}

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

	n := len(q.acquired)
	for id := range q.acquired {
		if err := q.Reject(ctx, id); err != nil {
			return err
		}
	}

	q.Logger.Debug(
		"loaded queue from journal",
		zap.Object("queue", (*logAdaptor)(q)),
		zap.Int("unacknowledged_count", n),
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
	enc.AddInt("size", a.queue.Len()+len(a.acquired))
	return nil
}
