package queue

import (
	"container/heap"
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Queue is a durable priority queue of messages.
type Queue struct {
	Journal journal.Journal[*JournalRecord]
	Key     func(m Message) string
	Logger  *zap.Logger

	version  uint32
	elements map[string]*elem
	queue    pqueue
	dangling map[string]*elem // acquired but unacknowledged messages, only used during load
}

// Message is a container for a message on a queue.
type Message struct {
	Envelope *envelopespec.Envelope
	MetaData proto.Message
}

// AcquiredMessage is a message that was acquired from a queue.
type AcquiredMessage struct {
	Envelope *envelopespec.Envelope
	MetaData proto.Message

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

	marshaled := q.marshalMessages(messages)
	if len(marshaled) == 0 {
		return nil
	}

	r := &JournalRecord_Enqueue{
		Enqueue: &EnqueueRecord{
			Messages: marshaled,
		},
	}

	if err := q.write(ctx, r); err != nil {
		return fmt.Errorf("unable to enqueue messages: %w", err)
	}

	q.applyEnqueue(marshaled)

	for _, m := range marshaled {
		q.Logger.Debug(
			"message enqueued",
			zap.String("key", m.Key),
			zapx.Envelope("message", m.Envelope),
			zap.Object("queue", (*logAdaptor)(q)),
		)
	}

	return nil
}

func (q *Queue) loadEnqueue(r *EnqueueRecord) {
	q.applyEnqueue(r.GetMessages())
}

func (q *Queue) applyEnqueue(messages []*JournalMessage) {
	for _, jm := range messages {
		e := &elem{JournalMessage: jm}
		q.elements[jm.Key] = e
		heap.Push(&q.queue, e)
	}
}

func (q *Queue) marshalMessages(messages []Message) []*JournalMessage {
	result := make([]*JournalMessage, 0, len(messages))

	makeKey := q.Key
	if makeKey == nil {
		makeKey = func(m Message) string {
			return m.Envelope.MessageId
		}
	}

	for _, m := range messages {
		if err := m.Envelope.Validate(); err != nil {
			panic(err)
		}

		key := makeKey(m)

		if _, ok := q.elements[key]; ok {
			q.Logger.Debug(
				"message ignored because it is already enqueued",
				zap.String("key", key),
				zapx.Envelope("message", m.Envelope),
				zap.Object("queue", (*logAdaptor)(q)),
			)
			continue
		}

		t, err := marshalkit.UnmarshalEnvelopeTime(m.Envelope.CreatedAt)
		if err != nil {
			panic(err)
		}

		var md *anypb.Any
		if m.MetaData != nil {
			var err error
			md, err = anypb.New(m.MetaData)
			if err != nil {
				panic(err)
			}
		}

		result = append(
			result,
			&JournalMessage{
				Envelope: m.Envelope,
				Key:      key,
				Priority: t.UnixNano(),
				MetaData: md,
			},
		)
	}

	return result
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
	if e.acquired {
		return AcquiredMessage{}, false, nil
	}

	m = AcquiredMessage{
		Envelope: e.Envelope,
		queue:    q,
		element:  e,
	}

	if e.MetaData != nil {
		md, err := e.MetaData.UnmarshalNew()
		if err != nil {
			return AcquiredMessage{}, false, fmt.Errorf("unable to unmarshal meta-data: %w", err)
		}
		m.MetaData = md
	}

	r := &JournalRecord_Acquire{
		Acquire: &AcquireRecord{
			Key: e.Key,
		},
	}

	if err := q.write(ctx, r); err != nil {
		return AcquiredMessage{}, false, fmt.Errorf("unable to acquire message: %w", err)
	}

	q.applyAcquire(e)

	q.Logger.Debug(
		"message acquired",
		zap.String("key", e.Key),
		zapx.Envelope("message", e.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return m, true, nil
}

func (q *Queue) loadAcquire(r *AcquireRecord) {
	e := q.elements[r.Key]
	q.applyAcquire(e)
	q.dangling[r.Key] = e
}

func (q *Queue) applyAcquire(e *elem) {
	e.acquired = true
	heap.Fix(&q.queue, e.index)
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

	if err := q.write(ctx, r); err != nil {
		return fmt.Errorf("unable to acknowledge message: %w", err)
	}

	q.applyAck(m.element)

	q.Logger.Debug(
		"message acknowledged",
		zap.String("key", m.element.Key),
		zapx.Envelope("message", m.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return nil
}

func (q *Queue) loadAck(r *AckRecord) {
	e := q.elements[r.Key]
	q.applyAck(e)
	delete(q.dangling, r.Key)
}

func (q *Queue) applyAck(e *elem) {
	q.elements[e.Key] = nil
	heap.Remove(&q.queue, e.index)
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

	if err := q.write(ctx, r); err != nil {
		return fmt.Errorf("unable to reject message: %w", err)
	}

	q.applyReject(m.element)

	q.Logger.Debug(
		"message rejected",
		zap.String("key", m.element.Key),
		zapx.Envelope("message", m.element.Envelope),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	return nil
}

func (q *Queue) loadReject(r *RejectRecord) {
	e := q.elements[r.Key]
	q.applyReject(e)
}

func (q *Queue) applyReject(e *elem) {
	e.acquired = false
	heap.Fix(&q.queue, e.index)
}

// load reads all entries from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.elements != nil {
		return nil
	}

	q.elements = map[string]*elem{}
	q.dangling = map[string]*elem{}

	for {
		r, ok, err := q.Journal.Read(ctx, q.version)
		if err != nil {
			return fmt.Errorf("unable to load queue: %w", err)
		}
		if !ok {
			break
		}

		r.GetOneOf().(journalRecord).load(q)
		q.version++
	}

	for _, e := range q.dangling {
		r := &JournalRecord_Reject{
			Reject: &RejectRecord{
				Key: e.Key,
			},
		}

		if err := q.write(ctx, r); err != nil {
			return fmt.Errorf("unable to load queue: %w", err)
		}
	}

	q.Logger.Debug(
		"loaded queue from journal",
		zap.Int("dangling_count", len(q.dangling)),
		zap.Object("queue", (*logAdaptor)(q)),
	)

	q.dangling = nil

	return nil
}

// write writes a record to the journal.
func (q *Queue) write(
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

	q.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	load(q *Queue)
}

func (x *JournalRecord_Enqueue) load(q *Queue) { q.loadEnqueue(x.Enqueue) }
func (x *JournalRecord_Acquire) load(q *Queue) { q.loadAcquire(x.Acquire) }
func (x *JournalRecord_Ack) load(q *Queue)     { q.loadAck(x.Ack) }
func (x *JournalRecord_Reject) load(q *Queue)  { q.loadReject(x.Reject) }

type logAdaptor Queue

func (a *logAdaptor) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint32("version", a.version)
	enc.AddInt("size", a.queue.Len())
	return nil
}
