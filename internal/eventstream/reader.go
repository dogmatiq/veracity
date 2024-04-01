package eventstream

import (
	"context"
	"fmt"
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
