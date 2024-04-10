package eventstream

import (
	"log/slog"

	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
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

	// canceled indicates that the [Supervisor] has canceled the subscription.
	canceled signaling.Event

	// id is a unique identifier for the subscriber.
	id *uuidpb.UUID

	// beginPos is the journal position to begin ranging or search for the next
	// event to deliver to the subscriber.
	beginPos journal.Position

	// beginPosIsDefinitive is true if beginPos "definitive", meaning that it
	// represents the exact position of the record containing the next event to
	// deliver to the subscriber.
	beginPosIsDefinitive bool
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
	defer w.resetIdleTimer()

	if sub.Offset >= w.nextOffset {
		w.Logger.Debug(
			"subscription activated, waiting for new events",
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
			slog.Int("subscriber_count", len(w.subscribers)),
			slog.String("subscriber_id", sub.id.AsString()),
			slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		)
		return
	}

	index := w.findInCache(sub.Offset)

	if index == -1 {
		delete(w.subscribers, sub)
		w.Logger.Warn(
			"subscription not activated due to request for uncached historical events",
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
			slog.Int("cached_event_count", len(w.recentEvents)),
			slog.Int("subscriber_count", len(w.subscribers)),
			slog.String("subscriber_id", sub.id.AsString()),
			slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		)
		sub.canceled.Signal()
		return
	}

	w.Logger.Debug(
		"subscription activated, delivering cached events",
		slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
		slog.Int("subscriber_count", len(w.subscribers)),
		slog.String("subscriber_id", sub.id.AsString()),
		slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
	)

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
		w.Logger.Debug(
			"subscription canceled by subscriber",
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
			slog.Int("subscriber_count", after),
			slog.String("subscriber_id", sub.id.AsString()),
			slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		)
		sub.canceled.Signal()
		w.resetIdleTimer()
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

	if sub.Filter != nil && !sub.Filter(event.Envelope) {
		sub.Offset++
		return eventFiltered
	}

	select {
	case sub.Events <- event:
		sub.Offset++
		return eventDelivered

	default:
		delete(w.subscribers, sub)
		w.Logger.Warn(
			"subscription canceled because the subscriber cannot keep up with the event stream",
			slog.Int("subscriber_count", len(w.subscribers)),
			slog.String("subscriber_id", sub.id.AsString()),
			slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		)
		sub.canceled.Signal()
		w.resetIdleTimer()

		return subscriptionCanceled
	}
}

// publishEvents publishes the events to both the recent event cache and any
// interested subscribers.
func (w *worker) publishEvents(pos journal.Position, rec *eventstreamjournal.Record) {
	op := rec.GetEventsAppended()
	if op == nil {
		return
	}

	skip := w.growCache(len(op.Events))

	// Update the subscriber's position to refer to the record containing the
	// events we're about to deliver.
	for sub := range w.subscribers {
		sub.beginPos = pos
		sub.beginPosIsDefinitive = true
	}

	offset := Offset(rec.StreamOffsetBefore)
	for i, env := range op.Events {
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
			slog.Int("subscriber_delivered_count", delivered),
			slog.Int("subscriber_filtered_count", filtered),
			slog.Int("subscriber_canceled_count", canceled),
		)
	}

	// Any remaining (i.e. uncanceled) subscribers should now look for the
	// following journal record.
	for sub := range w.subscribers {
		sub.beginPos++
	}
}
