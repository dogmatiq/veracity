package eventstream

import (
	"context"
	"log/slog"
	"time"

	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/signaling"
)

const defaultIdleTimeout = 5 * time.Minute

// A worker manages the state of an event stream.
type worker struct {
	// Journal stores the event stream's state.
	Journal journal.Journal[*eventstreamjournal.Record]

	// AppendQueue is a queue of requests to append events to the stream.
	AppendQueue messaging.ExchangeQueue[AppendRequest, AppendResponse]

	// Events is a channel to which events are published when they are appended
	// to the stream.
	Events chan<- Event

	// Shutdown signals the worker to stop when it next becomes idle.
	Shutdown signaling.Latch

	// IdleTimeout is the maximum amount of time the worker will sit idle before
	// shutting down. If it is non-positive, defaultIdleTimeout is used.
	IdleTimeout time.Duration

	// Logger is the target for log messages about the stream.
	Logger *slog.Logger

	pos journal.Position
	off Offset
}

// Run starts the worker.
//
// It processes requests until ctx is canceled, r.Shutdown is latched, or
// an error occurrs.
func (w *worker) Run(ctx context.Context) (err error) {
	w.Logger.DebugContext(ctx, "event stream worker started")
	defer w.Logger.DebugContext(ctx, "event stream worker stopped")

	pos, rec, ok, err := journal.LastRecord(ctx, w.Journal)
	if err != nil {
		return err
	}

	if ok {
		w.pos = pos + 1
		w.off = Offset(rec.StreamOffsetAfter)
	}

	return fsm.Start(ctx, w.idleState)
}

// idleState waits for a request or the shutdown signal.
func (w *worker) idleState(ctx context.Context) fsm.Action {
	duration := w.IdleTimeout
	if duration <= 0 {
		duration = defaultIdleTimeout
	}

	timeout := time.NewTimer(duration)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
		return fsm.Stop()

	case <-w.Shutdown.Signaled():
		return fsm.Stop()

	case <-timeout.C:
		return fsm.Stop()

	case ex := <-w.AppendQueue.Recv():
		return fsm.With(ex).EnterState(w.handleAppendState)
	}
}

// handleAppendState appends events to the stream.
func (w *worker) handleAppendState(
	ctx context.Context,
	ex messaging.Exchange[AppendRequest, AppendResponse],
) fsm.Action {
	n := len(ex.Request.Events)
	if n == 0 {
		panic("cannot record zero events")
	}

	res, err := w.appendEvents(ctx, ex.Request)
	if err != nil {
		ex.Err(err)
		return fsm.Fail(err)
	}

	ex.Ok(res)

	if res.AppendedByPriorAttempt {
		return fsm.EnterState(w.idleState)
	}

	return fsm.With2(ex.Request, res).EnterState(w.publishEventsState)
}

// publishEventsState publishes appended events to w.Events.
func (w *worker) publishEventsState(
	ctx context.Context,
	req AppendRequest,
	res AppendResponse,
) fsm.Action {
	for i, e := range req.Events {
		e := Event{
			Envelope: e,
			StreamID: req.StreamID,
			Offset:   res.BeginOffset + Offset(i),
		}

		select {
		case <-ctx.Done():
			return fsm.Stop()

		case w.Events <- e:
			continue
		}
	}

	return fsm.EnterState(w.idleState)
}

// appendEvents writes the events in req to the journal if they have not been
// written already. It returns the offset of the first event.
func (w *worker) appendEvents(
	ctx context.Context,
	req AppendRequest,
) (AppendResponse, error) {
	if w.mightBeDuplicates(req) {
		if rec, err := w.findAppendRecord(ctx, req); err == nil {
			for i, e := range req.Events {
				w.Logger.WarnContext(
					ctx,
					"ignored event that has already been appended to the stream",
					slog.Uint64("stream_offset", uint64(rec.StreamOffsetBefore)+uint64(i)),
					slog.String("message_id", e.MessageId.AsString()),
					slog.String("description", e.Description),
				)
			}

			return AppendResponse{
				BeginOffset:            Offset(rec.StreamOffsetBefore),
				EndOffset:              Offset(rec.StreamOffsetAfter),
				AppendedByPriorAttempt: true,
			}, nil
		} else if err != journal.ErrNotFound {
			return AppendResponse{}, err
		}
	}

	before := w.off
	after := w.off + Offset(len(req.Events))

	if err := w.Journal.Append(
		ctx,
		w.pos,
		eventstreamjournal.
			NewRecordBuilder().
			WithStreamOffsetBefore(uint64(before)).
			WithStreamOffsetAfter(uint64(after)).
			WithEventsAppended(&eventstreamjournal.EventsAppended{
				Events: req.Events,
			}).
			Build(),
	); err != nil {
		return AppendResponse{}, err
	}

	for i, e := range req.Events {
		w.Logger.InfoContext(
			ctx,
			"appended event to the stream",
			slog.Uint64("stream_offset", uint64(before)+uint64(i)),
			slog.String("message_id", e.MessageId.AsString()),
			slog.String("description", e.Description),
		)
	}

	w.pos++
	w.off = after

	return AppendResponse{
		BeginOffset:            before,
		EndOffset:              after,
		AppendedByPriorAttempt: false,
	}, nil
}

// mightBeDuplicates returns true if it's possible that the events in req have
// already been appended to the stream.
func (w *worker) mightBeDuplicates(req AppendRequest) bool {
	// The events can't be duplicates if there has never been a prior attempt to
	// append them.
	if req.IsFirstAttempt {
		return false
	}

	// The events can't be duplicates if the lowest possible offset that
	// they could have been appended is the current end of the stream.
	if req.LowestPossibleOffset == w.off {
		return false
	}

	return true
}

// findAppendRecord searches the journal to find the record that contains the
// append operation for the given events.
//
// TODO: This is a brute-force approach that searches the journal directly
// (though efficiently). We could improve upon this approach by keeping some
// in-memory state of recent event IDs (either explicitly, or via a bloom
// filter, for example).
func (w *worker) findAppendRecord(
	ctx context.Context,
	req AppendRequest,
) (*eventstreamjournal.Record, error) {
	return journal.ScanFromSearchResult(
		ctx,
		w.Journal,
		0,
		w.pos,
		eventstreamjournal.SearchByOffset(uint64(req.LowestPossibleOffset)),
		func(
			ctx context.Context,
			_ journal.Position,
			rec *eventstreamjournal.Record,
		) (*eventstreamjournal.Record, bool, error) {
			if op := rec.GetEventsAppended(); op != nil {
				targetID := req.Events[0].MessageId
				candidateID := op.Events[0].MessageId
				return rec, candidateID.Equal(targetID), nil
			}
			return nil, false, nil
		},
	)
}
