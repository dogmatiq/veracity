package eventstream

import (
	"context"
	"log/slog"
	"time"
)

const (
	// shutdownTimeout is the amount of time a worker WITH NO SUBSCRIBERS will
	// wait after appending events before shutting down.
	shutdownTimeout = 5 * time.Minute

	// catchUpTimeout is the amount of time a worker WITH SUBSCRIBERS will wait
	// after appending events before "catching up" with any journal records that
	// have been appended by other nodes.
	catchUpTimeout = 10 * time.Second
)

// resetIdleTimer starts or resets the idle timer.
func (w *worker) resetIdleTimer() {
	timeout := shutdownTimeout
	if len(w.subscribers) > 0 {
		timeout = catchUpTimeout
	}

	if w.idleTimer == nil {
		w.idleTimer = time.NewTimer(timeout)
	} else {
		if !w.idleTimer.Stop() {
			<-w.idleTimer.C
		}
		w.idleTimer.Reset(timeout)
	}
}

// handleIdle is called when the worker has not appended any new events for some
// period of time.
//
// If there are no subscribers, it returns false, indicating that the worker
// should shutdown. Otherwise, it reads the journal to see if there are new
// events to deliver to the subscribers.
func (w *worker) handleIdle(ctx context.Context) (bool, error) {
	if len(w.subscribers) == 0 {
		w.Logger.Debug(
			"event stream worker stopped due to inactivity",
			slog.Uint64("next_journal_position", uint64(w.nextPos)),
			slog.Uint64("next_stream_offset", uint64(w.nextOffset)),
		)
		return false, nil
	}

	if err := w.catchUpWithJournal(ctx); err != nil {
		return false, err
	}

	w.resetIdleTimer()

	return true, nil
}
