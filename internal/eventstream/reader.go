package eventstream

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/messaging"
)

// defaultSubscribeTimeout is the default maximum time to wait for a
// subscription to be acknowledged by the worker before reverting to reading
// from the journal.
const defaultSubscribeTimeout = 3 * time.Second

// A Reader reads ordered events from a stream.
type Reader struct {
	Journals         journal.BinaryStore
	SubscribeQueue   *messaging.ExchangeQueue[*Subscriber, messaging.None]
	UnsubscribeQueue *messaging.ExchangeQueue[*Subscriber, messaging.None]
	SubscribeTimeout time.Duration
	Logger           *slog.Logger
}

// Read sends events from a stream to the given subscriber's events channel.
//
// It first attempts to "sync" with the local worker to receive contemporary
// events in "real-time". If the subscriber's requested offset is too old to be
// obtained from the worker, or if the events channel becomes full, the reader
// obtains events directly from the journal until the last record is reached,
// then the process repeats.
func (r *Reader) Read(ctx context.Context, sub *Subscriber) error {
	if sub.id == nil {
		sub.id = uuidpb.Generate()
	}

	for {
		if err := r.readContemporary(ctx, sub); err != nil {
			return err
		}

		if err := r.readHistorical(ctx, sub); err != nil {
			return err
		}
	}
}

func (r *Reader) readContemporary(ctx context.Context, sub *Subscriber) error {
	// TODO: remote read

	r.Logger.Debug(
		"subscribing to receive contemporary events from local event stream worker",
		slog.String("stream_id", sub.StreamID.AsString()),
		slog.String("subscriber_id", sub.id.AsString()),
		slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
	)

	if err := r.subscribe(ctx, sub); err != nil {
		// If the subscription request times out, but the parent context isn't
		// canceled we revert to reading from the journal (by returning nil).
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			r.Logger.Warn(
				"timed-out waiting for local event stream worker to acknowledge subscription",
				slog.String("stream_id", sub.StreamID.AsString()),
				slog.String("subscriber_id", sub.id.AsString()),
				slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
				slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
			)
			return nil
		}

		return err
	}

	select {
	case <-ctx.Done():
		r.unsubscribe(sub)
		return ctx.Err()
	case <-sub.canceled.Signaled():
		r.Logger.Debug(
			"subscription canceled by local event stream worker",
			slog.String("stream_id", sub.StreamID.AsString()),
			slog.String("subscriber_id", sub.id.AsString()),
			slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
			slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		)
		return nil
	}
}

func (r *Reader) subscribe(ctx context.Context, sub *Subscriber) error {
	// Impose our own subscription timeout. This handles the case that the
	// supervisor/worker is not running or cannot service our subscription
	// request in a timely manner, in which case we will revert to reading from
	// the journal.
	timeout := r.SubscribeTimeout
	if timeout <= 0 {
		timeout = defaultSubscribeTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, done := r.SubscribeQueue.New(sub)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.SubscribeQueue.Send() <- req:
	}

	// We don't want to use the context timeout waiting for the response,
	// because then we wont know if the subscription was actually accepted. The
	// worker ALWAYS sends a response to any subscription request it receives
	// from the queue.
	res := <-done

	if _, err := res.Get(); err != nil {
		return fmt.Errorf("cannot subscribe to event stream: %w", err)
	}

	return nil
}

func (r *Reader) unsubscribe(sub *Subscriber) {
	// TODO: use a latch to indicate unsubscribing?
	req, _ := r.UnsubscribeQueue.New(sub)
	r.UnsubscribeQueue.Send() <- req
	<-sub.canceled.Signaled()
}

func (r *Reader) readHistorical(ctx context.Context, sub *Subscriber) error {
	j, err := eventstreamjournal.Open(ctx, r.Journals, sub.StreamID)
	if err != nil {
		return fmt.Errorf("unable to open journal: %w", err)
	}
	defer j.Close()

	var records, delivered, filtered int

	fn := func(
		ctx context.Context,
		pos journal.Position,
		rec *eventstreamjournal.Record,
	) (bool, error) {
		records++

		begin := Offset(rec.StreamOffsetBefore)
		end := Offset(rec.StreamOffsetAfter)

		if begin == end {
			// This condition is not producible at the moment, but is present to
			// provide forward compatibility with future journal record types.
			sub.beginPos = pos + 1
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

		sub.beginPos = pos
		sub.beginPosIsDefinitive = true

		index := sub.Offset - begin

		for _, env := range rec.GetEventsAppended().Events[index:] {
			if sub.Filter != nil && !sub.Filter(env) {
				sub.Offset++
				filtered++
				continue
			}

			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case sub.Events <- Event{sub.StreamID, sub.Offset, env}:
				sub.Offset++
				delivered++
			}
		}

		sub.beginPos++

		return true, nil
	}

	iter := r.searchHistorical
	if sub.beginPosIsDefinitive {
		iter = r.rangeHistorical
	}

	if err := iter(ctx, sub, j, fn); err != nil {
		return err
	}

	r.Logger.Debug(
		"finished reading historical events from journal",
		slog.String("stream_id", sub.StreamID.AsString()),
		slog.String("subscriber_id", sub.id.AsString()),
		slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		slog.Int("journal_record_count", records),
		slog.Int("events_delivered_count", delivered),
		slog.Int("events_filtered_count", filtered),
	)

	return nil
}

// rangeHistorical delivers all (relevent) events to the subscriber, starting
// with the events in the record at position sub.beginPos.
func (r *Reader) rangeHistorical(
	ctx context.Context,
	sub *Subscriber,
	j journal.Journal[*eventstreamjournal.Record],
	fn journal.RangeFunc[*eventstreamjournal.Record],
) error {
	r.Logger.Debug(
		"ranging over historical events in journal",
		slog.String("stream_id", sub.StreamID.AsString()),
		slog.String("subscriber_id", sub.id.AsString()),
		slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		slog.Uint64("journal_begin_position", uint64(sub.beginPos)),
	)

	if err := j.Range(ctx, sub.beginPos, fn); err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			// If we're ranging over a journal record that does not exist it
			// means we've delivered all events but were unable to subscribe to
			// receive contemporary events from the worker.
			return nil
		}
		return fmt.Errorf("unable to range over journal: %w", err)
	}

	return nil
}

// searchHistorical performs a binary search to find the journal record that
// contains the next event to deliver to sub, then delivers that event an all
// subsequent events until the end of the journal.
func (r *Reader) searchHistorical(
	ctx context.Context,
	sub *Subscriber,
	j journal.Journal[*eventstreamjournal.Record],
	fn journal.RangeFunc[*eventstreamjournal.Record],
) error {
	begin, end, err := j.Bounds(ctx)
	if err != nil {
		return fmt.Errorf("unable to read journal bounds: %w", err)
	}

	begin = max(begin, sub.beginPos)

	r.Logger.Debug(
		"searching for historical events in journal",
		slog.String("stream_id", sub.StreamID.AsString()),
		slog.String("subscriber_id", sub.id.AsString()),
		slog.Int("subscriber_headroom", cap(sub.Events)-len(sub.Events)),
		slog.Uint64("subscriber_stream_offset", uint64(sub.Offset)),
		slog.Uint64("journal_begin_position", uint64(begin)),
		slog.Uint64("journal_end_position", uint64(end)),
	)

	if err := journal.RangeFromSearchResult(
		ctx,
		j,
		begin, end,
		eventstreamjournal.SearchByOffset(uint64(sub.Offset)),
		fn,
	); err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			// If the event is not in the journal then we don't want to
			// re-search these same records in the future.
			sub.beginPos = end
			return nil
		}
		return fmt.Errorf("unable to search journal: %w", err)
	}

	return nil
}
