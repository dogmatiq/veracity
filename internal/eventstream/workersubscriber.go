package eventstream

import (
	"fmt"
	"log/slog"
)

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
