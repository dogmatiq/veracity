package eventstream

import (
	"context"
	"log/slog"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/signaling"
)

// A worker manages the state of an event stream.
type worker struct {
	// StreamID is the ID of the event stream that the worker manages.
	StreamID *uuidpb.UUID

	// Journal stores the event stream's state.
	Journal journal.Journal[*eventstreamjournal.Record]

	// AppendQueue is a queue of requests to append events to the stream.
	AppendQueue messaging.ExchangeQueue[AppendRequest, AppendResponse]

	// SubscribeQueue is a queue of requests to subscribe to the stream.
	SubscribeQueue messaging.RequestQueue[*Subscriber]

	// UnsubscribeQueue is a queue of requests to unsubscribe from the stream.
	UnsubscribeQueue messaging.RequestQueue[*Subscriber]

	// Shutdown signals the worker to stop when it next becomes idle.
	Shutdown signaling.Latch

	// Logger is the target for log messages about the stream.
	Logger *slog.Logger

	nextPos      journal.Position
	nextOffset   Offset
	recentEvents []Event
	idleTimer    *time.Timer
	subscribers  map[*Subscriber]struct{}
}

// Run starts the worker.
//
// It processes requests until ctx is canceled, an error occurs, the worker is
// shutdown by the supervisor, or the idle timeout expires.
func (w *worker) Run(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			w.Logger.Debug(
				"event stream worker stopped due to an error",
				slog.String("error", err.Error()),
				slog.Uint64("next_journal_position", uint64(w.nextPos)),
				slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
				slog.Int("subscriber_count", len(w.subscribers)),
			)
		}

		for sub := range w.subscribers {
			sub.canceled.Signal()
		}
	}()

	pos, rec, ok, err := journal.LastRecord(ctx, w.Journal)
	if err != nil {
		return err
	}

	if ok {
		w.nextPos = pos + 1
		w.nextOffset = Offset(rec.StreamOffsetAfter)

		w.Logger.Debug(
			"event stream journal has existing records",
			slog.Uint64("next_journal_position", uint64(w.nextPos)),
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
		)
	} else {
		w.Logger.Debug("event stream journal is empty")
	}

	w.resetIdleTimer()
	defer w.idleTimer.Stop()

	for {
		ok, err := w.tick(ctx)
		if !ok || err != nil {
			return err
		}
	}
}

// tick handles a single event stream operation.
func (w *worker) tick(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()

	case ex := <-w.AppendQueue.Recv():
		res, err := w.handleAppend(ctx, ex.Request)
		if err != nil {
			ex.Err(errShuttingDown)
		} else {
			ex.Ok(res)
		}
		return true, err

	case ex := <-w.SubscribeQueue.Recv():
		w.handleSubscribe(ex.Request)
		ex.Ok()
		return true, nil

	case ex := <-w.UnsubscribeQueue.Recv():
		w.handleUnsubscribe(ex.Request)
		ex.Ok()
		return true, nil

	case <-w.idleTimer.C:
		return w.handleIdle(ctx)

	case <-w.Shutdown.Signaled():
		w.Logger.Debug(
			"event stream worker stopped by supervisor",
			slog.Uint64("next_journal_position", uint64(w.nextPos)),
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
			slog.Int("subscriber_count", len(w.subscribers)),
		)
		return false, nil
	}
}

// catchUpWithJournal reads the journal to catch up with any records that have
// been appended by other nodes.
//
// It is called whenever the worker has some indication that it may be out of
// date, such as when there is an OCC conflict. It is also called periodically
// by otherwise idle workers.
func (w *worker) catchUpWithJournal(ctx context.Context) error {
	recordCount := 0
	eventCount := 0

	if err := w.Journal.Range(
		ctx,
		w.nextPos,
		func(
			ctx context.Context,
			pos journal.Position,
			rec *eventstreamjournal.Record,
		) (ok bool, err error) {
			recordCount++

			events := rec.GetEventsAppended().GetEvents()
			if len(events) != 0 {
				if eventCount == 0 {
					w.Logger.Warn("event stream contains events that were not appended by this worker")
				}
				w.publishEvents(Offset(rec.StreamOffsetBefore), events)
				eventCount += len(events)
			}

			w.nextPos = pos + 1
			w.nextOffset = Offset(rec.StreamOffsetAfter)

			return true, nil
		},
	); err != nil {
		return err
	}

	if recordCount != 0 {
		w.Logger.Debug(
			"caught up to the end of the event stream journal",
			slog.Int("record_count", recordCount),
			slog.Int("event_count", eventCount),
			slog.Uint64("next_journal_position", uint64(w.nextPos)),
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
		)
	}

	return nil
}
