package queue

import (
	"container/heap"
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/internal/protojournal"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Queue is a durable priority queue of messages.
type Queue struct {
	// Journal is the journal used to store the queue's state.
	Journal journal.BinaryJournal

	// DeriveIdempotencyKey is a function that derives the idempotency key to
	// use for each message on the queue.
	//
	// If it is nil the message's ID is used as the idempotency key.
	DeriveIdempotencyKey func(m Message) string

	// Logger is the target for log messages about changes to the queue.
	Logger *zap.Logger

	version    uint64
	elements   map[string]*elem
	queue      pqueue
	unreleased map[string]*elem // only used for recovery during load
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

// Enqueue adds message to the queue.
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
		return fmt.Errorf("unable to enqueue message(s): %w", err)
	}

	for _, m := range marshaled {
		q.enqueue(m)

		q.Logger.Debug(
			"message enqueued",
			zap.Uint64("queue_version", q.version),
			zap.Int("queue_size", q.queue.Len()),
			zap.String("idempotency_key", m.IdempotencyKey),
			zapx.Envelope("message", m.Envelope),
		)
	}

	return nil
}

// apply updates the queue's in-memory state to reflect an enqueue record.
func (x *JournalRecord_Enqueue) apply(q *Queue) {
	for _, jm := range x.Enqueue.GetMessages() {
		q.enqueue(jm)
	}
}

// enqueue updates the queue's in-memory state to include an enqueued message.
func (q *Queue) enqueue(jm *JournalMessage) {
	e := &elem{JournalMessage: jm}
	q.elements[jm.IdempotencyKey] = e
	heap.Push(&q.queue, e)
}

// useMessageIDAsKey is an idempotency key "derivation function" that uses the
// message's ID as the idempotency key.
func useMessageIDAsKey(m Message) string {
	return m.Envelope.MessageId
}

// marshalMessages marshals the in-memory representations of queued messages
// into their protocol buffers representation for storage in the journal.
//
// It ignores any messages that are already on the queue.
func (q *Queue) marshalMessages(messages []Message) []*JournalMessage {
	result := make([]*JournalMessage, 0, len(messages))

	deriveKey := q.DeriveIdempotencyKey
	if deriveKey == nil {
		deriveKey = useMessageIDAsKey
	}

	for _, m := range messages {
		if err := m.Envelope.Validate(); err != nil {
			panic(err)
		}

		createdAt, err := marshalkit.UnmarshalEnvelopeTime(m.Envelope.CreatedAt)
		if err != nil {
			panic(err)
		}

		key := deriveKey(m)

		if _, ok := q.elements[key]; ok {
			q.Logger.Debug(
				"ignored duplicate message",
				zap.String("idempotency_key", key),
				zapx.Envelope("message", m.Envelope),
			)

			continue
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
				Envelope:       m.Envelope,
				IdempotencyKey: key,
				Priority:       createdAt.UnixNano(),
				MetaData:       md,
			},
		)
	}

	return result
}

// Acquire returns the next message from the queue.
//
// If the queue is empty ok is false; otherwise, m is the next unacquired
// message in the queue.
//
// Acquiring a message from the queue does not immediately remove it from the
// queue. The message must subsequently be either released back onto the queue
// or removed entirely using Release() or Remove(), respectively.
func (q *Queue) Acquire(ctx context.Context) (m AcquiredMessage, ok bool, err error) {
	if err := q.load(ctx); err != nil {
		return AcquiredMessage{}, false, err
	}

	if q.queue.Len() == 0 {
		return AcquiredMessage{}, false, nil
	}

	e := q.queue.elements[0]
	if e.isAcquired {
		return AcquiredMessage{}, false, nil
	}

	m = AcquiredMessage{
		Envelope: e.Envelope,
		queue:    q,
		element:  e,
	}

	if e.MetaData != nil {
		var err error
		m.MetaData, err = e.MetaData.UnmarshalNew()
		if err != nil {
			return AcquiredMessage{}, false, fmt.Errorf("unable to unmarshal meta-data: %w", err)
		}
	}

	r := &JournalRecord_Acquire{
		Acquire: &AcquireRecord{
			IdempotencyKey: e.IdempotencyKey,
		},
	}

	if err := q.write(ctx, r); err != nil {
		return AcquiredMessage{}, false, fmt.Errorf("unable to acquire message: %w", err)
	}

	q.acquire(e)

	q.Logger.Debug(
		"message acquired from queue for processing",
		zap.Uint64("queue_version", q.version),
		zap.String("idempotency_key", e.IdempotencyKey),
		zapx.Envelope("message", e.Envelope),
	)

	return m, true, nil
}

// apply updates the queue's in-memory state to reflect an acquire record.
func (x *JournalRecord_Acquire) apply(q *Queue) {
	k := x.Acquire.IdempotencyKey
	e := q.elements[k]
	q.unreleased[k] = e

	q.acquire(e)
}

// acquire updates the queue's in-memory state to reflect acquisition of a
// message.
func (q *Queue) acquire(e *elem) {
	e.isAcquired = true
	heap.Fix(&q.queue, e.index)
}

// Release reverts an prior call to Acquire(), making the message eligible for
// re-acquisition.
func (q *Queue) Release(ctx context.Context, m AcquiredMessage) error {
	if m.queue != q || m.element == nil || !m.element.isAcquired {
		panic("message has not been acquired from this queue")
	}

	r := &JournalRecord_Release{
		Release: &ReleaseRecord{
			IdempotencyKey: m.element.IdempotencyKey,
		},
	}

	if err := q.write(ctx, r); err != nil {
		return fmt.Errorf("unable to release message: %w", err)
	}

	q.release(m.element)

	q.Logger.Debug(
		"message released for re-acquisition",
		zap.Uint64("queue_version", q.version),
		zap.String("idempotency_key", m.element.IdempotencyKey),
		zapx.Envelope("message", m.element.Envelope),
	)

	return nil
}

// apply updates the queue's in-memory state to reflect a release record.
func (x *JournalRecord_Release) apply(q *Queue) {
	k := x.Release.IdempotencyKey
	e := q.elements[k]

	q.release(e)
}

// release updates the queue's in-memory state to reflect releasing of an
// acquired message.
func (q *Queue) release(e *elem) {
	e.isAcquired = false
	heap.Fix(&q.queue, e.index)
}

// load reads all records from the journal and applies them to the queue.
func (q *Queue) load(ctx context.Context) error {
	if q.elements != nil {
		return nil
	}

	q.elements = map[string]*elem{}
	q.unreleased = map[string]*elem{}

	rec := &JournalRecord{}

	for {
		ok, err := protojournal.Read(ctx, q.Journal, q.version, rec)
		if err != nil {
			return fmt.Errorf("unable to load queue: %w", err)
		}
		if !ok {
			break
		}

		rec.GetOneOf().(journalRecord).apply(q)
		q.version++
	}

	for _, e := range q.unreleased {
		rec := &JournalRecord_Release{
			Release: &ReleaseRecord{
				IdempotencyKey: e.IdempotencyKey,
			},
		}

		if err := q.write(ctx, rec); err != nil {
			return fmt.Errorf("unable to release message: %w", err)
		}
	}

	q.Logger.Debug(
		"loaded queue",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.queue.Len()),
		zap.Int("released_message_count", len(q.unreleased)),
	)

	q.unreleased = nil

	return nil
}

// Remove permanently removes a message from the queue.
func (q *Queue) Remove(ctx context.Context, m AcquiredMessage) error {
	if m.queue != q || m.element == nil || !m.element.isAcquired {
		panic("message has not been acquired from this queue")
	}

	r := &JournalRecord_Remove{
		Remove: &RemoveRecord{
			IdempotencyKey: m.element.IdempotencyKey,
		},
	}

	if err := q.write(ctx, r); err != nil {
		return fmt.Errorf("unable to remove message: %w", err)
	}

	q.remove(m.element)

	q.Logger.Debug(
		"message removed from queue",
		zap.Uint64("queue_version", q.version),
		zap.Int("queue_size", q.queue.Len()),
		zap.String("idempotency_key", m.element.IdempotencyKey),
		zapx.Envelope("message", m.Envelope),
	)

	return nil
}

// apply updates the queue's in-memory state to reflect a remove record.
func (x *JournalRecord_Remove) apply(q *Queue) {
	k := x.Remove.IdempotencyKey
	e := q.elements[k]
	delete(q.unreleased, k)

	q.remove(e)
}

// remove updates the queue's in-memory state to reflect removal of a message.
func (q *Queue) remove(e *elem) {
	q.elements[e.IdempotencyKey] = nil
	heap.Remove(&q.queue, e.index)
}

// write writes a record to the journal.
func (q *Queue) write(ctx context.Context, r journalRecord) error {
	ok, err := protojournal.Write(
		ctx,
		q.Journal,
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
	apply(*Queue)
}
