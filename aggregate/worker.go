package aggregate

import (
	"context"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/linger"
)

// DefaultIdleTimeout is the default amount of time a worker will continue
// running without receiving a command.
const DefaultIdleTimeout = 5 * time.Minute

// WorkerConfig encapsulates the configuration and dependencies of a worker.
type WorkerConfig struct {
	// Handler is the message handler that handles command messages routed to
	// this instance.
	Handler dogma.AggregateMessageHandler

	// Identity is the identity of the handler.
	HandlerIdentity configkit.Identity

	// Loader is used to load aggregate state from persistent storage.
	Loader *Loader

	// EventWriter is used to write new events to persistent storage.
	EventWriter EventWriter

	// SnapshotWriter is used to persist snapshots of the aggregate root.
	SnapshotWriter SnapshotWriter

	// IdleTimeout is the amount of time a worker will continue running without
	// receiving a command.
	IdleTimeout time.Duration

	// SnapshotInterval is the maximum number of events that can be recorded
	// before a new snapshot is taken.
	//
	// If it is 0, DefaultSnapshotInterval is used.
	SnapshotInterval uint64

	// SnapshotWriteTimeout is the maximum duration to allow for a snapshot to
	// be written to persistent storage.
	//
	// If it is non-positive, DefaultSnapshotWriteTimeout is used instead.
	SnapshotWriteTimeout time.Duration

	// Logger is the target for log messages about the aggregate instance.
	Logger logging.Logger
}

// Worker manages the lifecycle of a single aggregate instance.
type Worker struct {
	WorkerConfig

	// InstanceID is the instance of the aggregate managed by this worker.
	InstanceID string

	// Commands is a channel that receives commands to be executed.
	Commands <-chan Command

	// root is the aggregate root for this instance.
	root dogma.AggregateRoot

	// firstOffset is the offset of the first non-archived event.
	firstOffset uint64

	// nextOffset is the offset of the next event that will be recorded by this
	// instance.
	nextOffset uint64

	// snapshotAge is the number of events that have been recorded since the
	// last snapshot was taken.
	snapshotAge uint64
}

// Run starts the worker until ctx is canceled or the worker exits due to an
// idle timeout.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.loadRoot(ctx); err != nil {
		return err
	}

	for {
		if done, err := w.handleNextCommand(ctx); done || err != nil {
			return err
		}
	}
}

// loadRoot loads the aggregate root, populating w.root, w.firstOffset,
// w.nextOffset and w.snapshotAge.
func (w *Worker) loadRoot(ctx context.Context) error {
	var err error
	w.root = w.Handler.New()

	w.firstOffset, w.nextOffset, w.snapshotAge, err = w.Loader.Load(
		ctx,
		w.HandlerIdentity,
		w.InstanceID,
		w.root,
	)

	return err
}

// handleNextCommand waits for the next incoming command, then handles it.
func (w *Worker) handleNextCommand(ctx context.Context) (done bool, _ error) {
	idle := time.NewTimer(
		linger.MustCoalesce(
			w.IdleTimeout,
			DefaultIdleTimeout,
		),
	)
	defer idle.Stop()

	select {
	case <-ctx.Done():
		return true, ctx.Err()

	case cmd := <-w.Commands:
		return w.handleCommand(ctx, cmd)

	case <-idle.C:
		// We've been idle for a while without receiving a command.
		//
		// We always take a snapshot before shutting down to mininize the time
		// it takes to reload this aggregate in the future.
		return true, w.takeSnapshot(ctx)
	}
}

// handleCommand handles a single command.
func (w *Worker) handleCommand(
	ctx context.Context,
	cmd Command,
) (done bool, _ error) {
	sc := &scope{}

	w.Handler.HandleCommand(
		w.root,
		sc,
		cmd.Parcel.Message,
	)

	if err := w.EventWriter.WriteEvents(
		ctx,
		w.HandlerIdentity.Key,
		w.InstanceID,
		w.firstOffset,
		w.nextOffset,
		sc.EventEnvelopes,
		sc.IsDestroyed, // archive events if instance is destroyed
	); err != nil {
		return true, err
	}

	// We are only responsible for writing "success" responses, all error
	// responses are handled by the supervisor.
	cmd.Result <- nil

	w.snapshotAge += uint64(len(sc.EventEnvelopes))
	w.nextOffset += uint64(len(sc.EventEnvelopes))

	if sc.IsDestroyed {
		return true, w.archiveSnapshots(ctx)
	}

	return false, w.takeSnapshotIfIntervalExceeded(ctx)
}

// takeSnapshotIfIntervalExceeded takes a new snapshot of the aggregate state if
// the most recent snapshot's age exceeds the configured snapshot interval.
//
// A failure to persist a snapshot is not treated as an error; err is non-nil
// only if ctx is canceled.
func (w *Worker) takeSnapshotIfIntervalExceeded(ctx context.Context) error {
	interval := w.SnapshotInterval
	if interval == 0 {
		interval = DefaultSnapshotInterval
	}

	if w.snapshotAge < interval {
		return nil
	}

	return w.takeSnapshot(ctx)
}

// takeSnapshot takes a new snapshot of the aggregate state.
//
// A failure to persist a snapshot is not treated as an error; err is non-nil
// only if ctx is canceled.
func (w *Worker) takeSnapshot(ctx context.Context) error {
	// Snapshots are entirely optional, bail if no writer is configured.
	if w.SnapshotWriter == nil {
		return nil
	}

	// If the snapshotAge is zero then the current snapshot is already
	// persisted.
	if w.snapshotAge == 0 {
		return nil
	}

	timeoutCtx, cancel := linger.ContextWithTimeout(
		ctx,
		w.SnapshotWriteTimeout,
		DefaultSnapshotWriteTimeout,
	)
	defer cancel()

	snapshotOffset := w.nextOffset - 1

	if err := w.SnapshotWriter.WriteSnapshot(
		timeoutCtx,
		w.HandlerIdentity.Key,
		w.InstanceID,
		w.root,
		snapshotOffset,
	); err != nil {
		// If the error was due to a context cancelation/timeout of the PARENT
		// content we bail with the context error.
		if err == ctx.Err() {
			return err
		}

		logging.Log(
			w.Logger,
			"up-to-date snapshot of aggregate root %s[%s] cannot be be written at offset %d: %w",
			w.HandlerIdentity.Name,
			w.InstanceID,
			snapshotOffset,
			err,
		)

		return nil
	}

	w.snapshotAge = 0

	logging.Log(
		w.Logger,
		"up-to-date snapshot of aggregate root %s[%s] written at offset %d: %w",
		w.HandlerIdentity.Name,
		w.InstanceID,
		snapshotOffset,
	)

	return nil
}

// archiveSnapshots archives any existing snapshots of this instance.
//
// A failure to archive snapshots is not treated as an error; err is non-nil
// only if ctx is canceled. A failed attempt at archiving is never retried.
func (w *Worker) archiveSnapshots(ctx context.Context) error {
	if err := w.SnapshotWriter.ArchiveSnapshots(
		ctx,
		w.HandlerIdentity.Key,
		w.InstanceID,
	); err != nil {
		// If the error was due to a context cancelation/timeout we bail with
		// the context error.
		if err == ctx.Err() {
			return err
		}

		logging.Log(
			w.Logger,
			"snapshots of aggregate root %s[%s] cannot be be archived: %w",
			w.HandlerIdentity.Name,
			w.InstanceID,
		)
	}

	return nil
}
