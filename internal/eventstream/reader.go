package eventstream

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/signaling"
)

// A Subscriber is sent events from a stream, by way of a [Reader].
type Subscriber struct {
	// StreamID is the ID of the stream from which events are read.
	StreamID *uuidpb.UUID

	// Offset is the offset of the next event to read.
	//
	// It must not be read or modified while the subscription is active. It is
	// incremented as events are sent to the subscriber.
	Offset Offset

	// Filter is a predicate function that returns true if the subscriber should
	// receive the event in the given envelope.
	//
	// It is used to avoid filling the subscriber's channel with events they are
	// not interested in. It is called by the event stream worker in its own
	// goroutine, and hence must not block.
	Filter func(*envelopepb.Envelope) bool

	// Events is the channel to which the subscriber's events are sent.
	Events chan<- Event

	canceled signaling.Event
}

// A Reader reads ordered events from a stream.
type Reader struct {
	Journals         journal.BinaryStore
	SubscribeQueue   *messaging.RequestQueue[*Subscriber]
	UnsubscribeQueue *messaging.RequestQueue[*Subscriber]
}

// Read reads events from a stream and sends them to the given subscriber.
//
// It starts by reading events directly from the stream's journal records. Once
// it has "caught up" to the end of the journal it receives events in
// "real-time" from the supervisor of that stream.
//
// If the subscriber's channel becomes full, it reverts to reading from the
// journal until it catches up again.
func (r *Reader) Read(ctx context.Context, sub *Subscriber) error {
	for {
		if err := r.readHistorical(ctx, sub); err != nil {
			return err
		}

		if err := r.readContemporary(ctx, sub); err != nil {
			return err
		}
	}
}

func (r *Reader) readHistorical(ctx context.Context, sub *Subscriber) error {
	j, err := eventstreamjournal.Open(ctx, r.Journals, sub.StreamID)
	if err != nil {
		return err
	}
	defer j.Close()

	searchBegin, searchEnd, err := j.Bounds(ctx)
	if err != nil {
		return err
	}

	return journal.RangeFromSearchResult(
		ctx,
		j,
		searchBegin, searchEnd,
		eventstreamjournal.SearchByOffset(uint64(sub.Offset)),
		func(
			ctx context.Context,
			pos journal.Position,
			rec *eventstreamjournal.Record,
		) (bool, error) {
			begin := Offset(rec.StreamOffsetBefore)
			end := Offset(rec.StreamOffsetAfter)

			if begin == end {
				// no events in this record
				return true, nil
			}

			if sub.Offset < begin || sub.Offset >= end {
				return false, fmt.Errorf(
					"event stream integrity error at journal position %d: expected event at offset %d, but found offset range [%d, %d)",
					pos,
					sub.Offset,
					begin,
					end,
				)
			}

			index := sub.Offset - begin

			for _, env := range rec.GetEventsAppended().Events[index:] {
				if !sub.Filter(env) {
					sub.Offset++
					continue
				}

				select {
				case <-ctx.Done():
					return false, ctx.Err()
				case sub.Events <- Event{sub.StreamID, sub.Offset, env}:
					sub.Offset++
				}
			}

			return true, nil
		},
	)
}

func (r *Reader) readContemporary(ctx context.Context, sub *Subscriber) error {
	// TODO: remote read

	if err := r.subscribe(ctx, sub); err != nil {
		return err
	}
	defer r.unsubscribe(ctx, sub)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sub.canceled.Signaled():
		return nil
	}
}

func (r *Reader) subscribe(ctx context.Context, sub *Subscriber) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // TODO: make configurable
	cancel()

	if err := r.SubscribeQueue.Do(ctx, sub); err != nil {
		return fmt.Errorf("cannot subscribe to event stream: %w", err)
	}

	return nil
}

func (r *Reader) unsubscribe(ctx context.Context, sub *Subscriber) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	// Cancel the unsubscribe context when the subscription is canceled,
	// regardless of the reason.
	//
	// This handles the situation where the subscription is canceled because the
	// worker shutdown (and hence wont service the unsubscribe request).
	go func() {
		<-sub.canceled.Signaled()
		cancel()
	}()

	return r.UnsubscribeQueue.Do(ctx, sub)
}

// handleSubscribe adds sub to the subscriber list.
//
// It delivers any cached events that the subscriber has not yet seen. If the
// subscriber's requested event is older than the events in the cache the
// subscription is canceled immediately.
func (w *worker) handleSubscribe(sub *Subscriber) {
	if !sub.StreamID.Equal(w.StreamID) {
		panic("received request for a different stream ID")
	}

	if w.subscribers == nil {
		w.subscribers = map[*Subscriber]struct{}{}
	}
	w.subscribers[sub] = struct{}{}

	w.Logger.Debug(
		"subscription activated",
		slog.String("channel_address", fmt.Sprint(sub.Events)),
		slog.Int("channel_capacity", cap(sub.Events)),
		slog.Int("channel_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Int("subscriber_count", len(w.subscribers)),
		slog.Uint64("requested_stream_offset", uint64(sub.Offset)),
		slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
	)

	if sub.Offset >= w.nextOffset {
		return
	}

	index := w.findInCache(sub.Offset)

	if index == -1 {
		sub.canceled.Signal()
		w.Logger.Warn(
			"subscription canceled immediately due request for historical events",
			slog.String("channel_address", fmt.Sprint(sub.Events)),
			slog.Int("channel_capacity", cap(sub.Events)),
			slog.Int("channel_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Int("subscriber_count", len(w.subscribers)),
			slog.Int("cached_event_count", len(w.recentEvents)),
			slog.Uint64("requested_stream_offset", uint64(sub.Offset)),
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
		)
		return
	}

	for _, event := range w.recentEvents[index:] {
		if w.deliverEventToSubscriber(event, sub) == subscriptionCanceled {
			return
		}
	}
}

// handleUnsubscribe removes sub from the subscriber list.
func (w *worker) handleUnsubscribe(sub *Subscriber) {
	if !sub.StreamID.Equal(w.StreamID) {
		panic("received request for a different stream ID")
	}

	before := len(w.subscribers)
	delete(w.subscribers, sub)
	after := len(w.subscribers)

	if before > after {
		sub.canceled.Signal()

		w.Logger.Debug(
			"subscription canceled by subscriber",
			slog.String("channel_address", fmt.Sprint(sub.Events)),
			slog.Int("channel_capacity", cap(sub.Events)),
			slog.Int("channel_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Int("subscriber_count", after),
		)
	}
}

// deliverResult is an enumeration of the possible outcomes of delivering an
// event to a subscriber.
type deliverResult int

const (
	// eventDelivered means that the event was sent to the subscriber's events
	// channel, which may or may not be buffered.
	eventDelivered deliverResult = iota

	// eventFiltered means that the event was filtered by the subscriber's
	// filter function, and did not need to be delivered.
	eventFiltered

	// subscriptionCanceled means that an attempt was made to send the event to
	// the subscriber's event channel, but the channel buffer was full (or
	// unbuffered and not ready to read), and so the subscription was canceled.
	subscriptionCanceled
)

// deliverEventToSubscriber attempts to deliver an event to a subscriber's event
// channel.
func (w *worker) deliverEventToSubscriber(event Event, sub *Subscriber) deliverResult {
	if event.Offset > sub.Offset {
		panic("event is out of order")
	}

	if event.Offset < sub.Offset {
		return eventFiltered
	}

	if !sub.Filter(event.Envelope) {
		sub.Offset++
		return eventFiltered
	}

	select {
	case sub.Events <- event:
		sub.Offset++
		return eventDelivered

	default:
		delete(w.subscribers, sub)
		sub.canceled.Signal()

		w.Logger.Warn(
			"subscription canceled because the subscriber can not keep up with the event stream",
			slog.String("channel_address", fmt.Sprint(sub.Events)),
			slog.Int("channel_capacity", cap(sub.Events)),
			slog.Int("channel_headroom", 0),
			slog.Int("subscriber_count", len(w.subscribers)),
			slog.Uint64("stream_offset", uint64(event.Offset)),
		)

		return subscriptionCanceled
	}
}

// publishEvents publishes the events to both the recent event cache and any
// interested subscribers.
func (w *worker) publishEvents(
	offset Offset,
	events []*envelopepb.Envelope,
) {
	skip := w.growCache(len(events))

	for i, env := range events {
		event := Event{w.StreamID, offset, env}
		offset++

		if i >= skip {
			w.appendEventToCache(event)
		}

		if len(w.subscribers) == 0 {
			continue
		}

		var delivered, filtered, canceled int

		for sub := range w.subscribers {
			switch w.deliverEventToSubscriber(event, sub) {
			case eventDelivered:
				delivered++
			case eventFiltered:
				filtered++
			case subscriptionCanceled:
				canceled++
			}
		}

		w.Logger.Debug(
			"event published to subscribers",
			slog.Uint64("stream_offset", uint64(event.Offset)),
			slog.String("message_id", env.MessageId.AsString()),
			slog.String("description", env.Description),
			slog.Int("delivered_count", delivered),
			slog.Int("filtered_count", filtered),
			slog.Int("canceled_count", canceled),
		)
	}
}
