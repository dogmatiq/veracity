package eventstream

import (
	"context"
	"log/slog"

	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
)

// AppendRequest is a request to append events to an event stream.
type AppendRequest struct {
	// StreamID is the ID of the event stream to which the events are appended.
	StreamID *uuidpb.UUID

	// Events is the set of events to append to the stream. It must not be
	// empty.
	Events []*envelopepb.Envelope

	// LowestPossibleOffset is the lowest offset within the stream at which
	// these events may have already been appended.
	LowestPossibleOffset Offset
}

// AppendResponse is the successful result of an [AppendRequest].
type AppendResponse struct {
	// [BeginOffset, EndOffset) is the half-open range describing the offsets of the
	// appended events within the stream. That is, BeginOffset is the offset of the
	// first event in the [AppendRequest], and EndOffset is the offset after the last
	// event in the [AppendRequest].
	BeginOffset, EndOffset Offset

	// AppendedByPriorAttempt is true if the events were appended by a prior
	// [AppendRequest] and hence deduplicated.
	AppendedByPriorAttempt bool
}

func (w *worker) handleAppend(
	ctx context.Context,
	req AppendRequest,
) (AppendResponse, error) {
	if !req.StreamID.Equal(w.StreamID) {
		panic("received request for a different stream ID")
	}

	if len(req.Events) == 0 {
		// We panic rather than just failing the exchange because we never want
		// empty requests to occupy space in the worker's queue. The sender
		// should simply not send empty requests.
		panic("received append request with no events")
	}

	defer w.resetIdleTimer()

	if req.LowestPossibleOffset > w.nextOffset {
		if err := w.catchUpWithJournal(ctx); err != nil {
			return AppendResponse{}, err
		}
	}

	for {
		res, err := w.findPriorAppend(ctx, req)
		if err != nil {
			return AppendResponse{}, err
		}

		if res.AppendedByPriorAttempt {
			for index, event := range req.Events {
				w.Logger.Info(
					"discarded duplicate event",
					slog.Uint64("stream_offset", uint64(res.BeginOffset)+uint64(index)),
					slog.String("message_id", event.MessageId.AsString()),
					slog.String("description", event.Description),
				)
			}
			return res, nil
		}

		res, err = w.writeEventsToJournal(ctx, req)
		if err == nil {
			w.publishEvents(res.BeginOffset, req.Events)
			return res, nil
		}

		if err != journal.ErrConflict {
			return AppendResponse{}, err
		}

		if err := w.catchUpWithJournal(ctx); err != nil {
			return AppendResponse{}, err
		}
	}
}

// findPriorAppend returns an [AppendResponse] if the given [AppendRequest] has
// already been handled.
func (w *worker) findPriorAppend(
	ctx context.Context,
	req AppendRequest,
) (AppendResponse, error) {
	// If the lowest possible offset is ahead of the next offset the request is
	// malformed. Either theres a bug in Veracity, or the journal has suffered
	// catastrophic data loss.
	if req.LowestPossibleOffset > w.nextOffset {
		panic("lowest possible offset is greater than the next offset")
	}

	// If the lowest possible offset is equal to the next offset, no events
	// have been recorded since the the request was created, and hence there
	// can be no prior append attempt.
	if req.LowestPossibleOffset == w.nextOffset {
		return AppendResponse{}, nil
	}

	// If the lowest possible offset is in the cache, we can check for
	// duplicates without using the journal. We search using the last event in
	// the request as it's the most likely to still be in the cache.
	lowestPossibleOffset := req.LowestPossibleOffset + Offset(len(req.Events))

	if cacheIndex := w.findInCache(lowestPossibleOffset); cacheIndex != -1 {
		lastMessageIndex := len(req.Events) - 1
		lastMessageID := req.Events[lastMessageIndex].MessageId

		for _, event := range w.recentEvents[cacheIndex:] {
			if event.Envelope.MessageId.Equal(lastMessageID) {
				return AppendResponse{
					// We know the offset of the last message in the request, so
					// we can compute the offset of the first message, even if
					// it's no longer in the cache.
					BeginOffset:            event.Offset - Offset(lastMessageIndex),
					EndOffset:              event.Offset + 1,
					AppendedByPriorAttempt: true,
				}, nil
			}
		}
	}

	// Finally, we search the journal for the record containing the events.
	rec, err := journal.ScanFromSearchResult(
		ctx,
		w.Journal,
		0,
		w.nextPos,
		eventstreamjournal.SearchByOffset(uint64(req.LowestPossibleOffset)),
		func(
			_ context.Context,
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
	if err != nil {
		return AppendResponse{}, journal.IgnoreNotFound(err)
	}

	return AppendResponse{
		BeginOffset:            Offset(rec.StreamOffsetBefore),
		EndOffset:              Offset(rec.StreamOffsetAfter),
		AppendedByPriorAttempt: true,
	}, nil
}

func (w *worker) writeEventsToJournal(
	ctx context.Context,
	req AppendRequest,
) (AppendResponse, error) {
	before := w.nextOffset
	after := w.nextOffset + Offset(len(req.Events))

	if err := w.Journal.Append(
		ctx,
		w.nextPos,
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

	for index, event := range req.Events {
		w.Logger.Info(
			"appended event to the stream",
			slog.Uint64("journal_position", uint64(w.nextPos)),
			slog.Uint64("stream_offset", uint64(before)+uint64(index)),
			slog.String("message_id", event.MessageId.AsString()),
			slog.String("description", event.Description),
		)
	}

	w.nextPos++
	w.nextOffset = after

	return AppendResponse{
		BeginOffset:            before,
		EndOffset:              after,
		AppendedByPriorAttempt: false,
	}, nil
}
