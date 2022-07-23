package aggregate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/parcel"
)

// DefaultIdleTimeout is the default amount of time a worker will continue
// running without receiving a command.
const DefaultIdleTimeout = 5 * time.Minute

// WorkerConfig encapsulates the configuration and dependencies of a worker.
type WorkerConfig struct {
	// Handler is the message handler that handles command messages routed to
	// this instance.
	Handler dogma.AggregateMessageHandler

	// HandlerIdentity is the identity of the handler.
	HandlerIdentity configkit.Identity

	// Packer is used to create parcels containing the recorded events.
	Packer *parcel.Packer

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

	// Logger is the target for log messages about the aggregate instance.
	Logger logging.Logger
}

// Worker manages the lifecycle of a single aggregate instance.
type Worker struct {
	WorkerConfig

	// InstanceID is the instance of the aggregate managed by this worker.
	InstanceID string

	// Commands is a channel that receives commands to be executed.
	Commands <-chan *Command

	// Idle is used to signal that the worker has become idle by writing the
	// worker's instance ID to the channel.
	//
	// The supervisor may stop the worker by closing the commands channel or
	// canceling the context passed to Run(), or it may choose to ignore the
	// idle signal and keep sending commands.
	Idle chan<- string

	// envHandlerIdentity is the identity of the handler in the representation
	// used within envelopes.
	envHandlerIdentity *envelopespec.Identity

	// root is the aggregate root for this instance.
	root dogma.AggregateRoot

	// [begin, end) is the range of unarchived revisions for this instance.
	begin, end uint64

	// snapshotAge is the number of revisions that have been made since the last
	// snapshot was taken.
	snapshotAge uint64

	// currentState is the current state of the worker.
	currentState workerState
}

// errShutdown is an error that indicates a worker is shutting down.
var errShutdown = errors.New("shutting down")

// workerState is a function that provides worker logic for a specific state.
//
// It returns the next state that the worker should transition to, or nil to
// indicate that the worker is done.
type workerState func(ctx context.Context) (workerState, error)

// Run handles messages that are written to the worker's commands channel.
//
// It returns when ctx is canceled, an error occurs, or the commands channel is
// closed.
func (w *Worker) Run(ctx context.Context) error {
	w.envHandlerIdentity = marshalkit.MustMarshalEnvelopeIdentity(w.HandlerIdentity)
	w.currentState = w.stateLoadRoot

	for w.currentState != nil {
		var err error
		w.currentState, err = w.currentState(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// stateLoadRoot loads the aggregate root.
//
// It populates w.root, w.begin, w.end and w.snapshotAge.
func (w *Worker) stateLoadRoot(ctx context.Context) (workerState, error) {
	var err error
	w.root = w.Handler.New()

	w.begin, w.end, w.snapshotAge, err = w.Loader.Load(
		ctx,
		w.HandlerIdentity,
		w.InstanceID,
		w.root,
	)

	if err != nil {
		return nil, err
	}

	return w.stateWaitForCommand, nil
}

// stateWaitForCommand blocks until a command is available for handling, or the
// idle timeout is exceeded.
func (w *Worker) stateWaitForCommand(ctx context.Context) (workerState, error) {
	idle := time.NewTimer(
		linger.MustCoalesce(
			w.IdleTimeout,
			DefaultIdleTimeout,
		),
	)
	defer idle.Stop()

	select {
	case cmd, ok := <-w.Commands:
		return w.stateHandleCommand(cmd, ok), nil

	case <-idle.C:
		return w.stateRequestShutdown, w.takeSnapshot(ctx)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// stateHandleCommand calls the user-defined message handler for a single
// command, then persists the changes it makes.
func (w *Worker) stateHandleCommand(cmd *Command, ok bool) workerState {
	if !ok {
		return nil
	}

	return func(ctx context.Context) (workerState, error) {
		sc := &scope{
			Command:         cmd.Parcel,
			HandlerIdentity: w.envHandlerIdentity,
			ID:              w.InstanceID,
			Root:            w.root,
			Packer:          w.Packer,
			IsDestroyed:     w.begin >= w.end,
			Logger: logging.Prefix(
				w.Logger,
				"aggregate %s[%s]: ",
				w.HandlerIdentity.Name,
				w.InstanceID,
			),
		}

		w.Handler.HandleCommand(
			w.root,
			sc,
			cmd.Parcel.Message,
		)

		err := w.saveChanges(ctx, sc)
		if err != nil {
			cmd.Nack(errShutdown)
			return nil, err
		}

		cmd.Ack()

		if sc.IsDestroyed {
			return w.stateRequestShutdown, w.archiveSnapshots(ctx)
		}

		return w.stateWaitForCommand, w.takeSnapshotIfIntervalExceeded(ctx)
	}
}

// stateRequestShutdown signals that the worker is is idle and would like to
// shutdown.
//
// It blocks until the supervisor receives the shutdown request or a command is
// received.
func (w *Worker) stateRequestShutdown(ctx context.Context) (workerState, error) {
	select {
	case cmd, ok := <-w.Commands:
		return w.stateHandleCommand(cmd, ok), nil

	case w.Idle <- w.InstanceID:
		return w.stateWaitForCommand, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// saveChanges persists changes made as a result of handling a single command.
func (w *Worker) saveChanges(ctx context.Context, sc *scope) error {
	if !sc.HasChanges {
		return nil
	}

	begin := w.begin
	if sc.IsDestroyed {
		begin = w.end + 1
	}

	if err := w.EventWriter.WriteEvents(
		ctx,
		w.HandlerIdentity.Key,
		w.InstanceID,
		begin,
		w.end,
		sc.EventEnvelopes,
	); err != nil {
		return fmt.Errorf(
			"cannot write events for aggregate root %s[%s]: %w",
			w.HandlerIdentity.Name,
			w.InstanceID,
			err,
		)
	}

	w.begin = begin
	w.end++
	w.snapshotAge++

	return nil
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

	snapshotRev := w.end - 1

	if err := w.SnapshotWriter.WriteSnapshot(
		ctx,
		w.HandlerIdentity.Key,
		w.InstanceID,
		w.root,
		snapshotRev,
	); err != nil {
		// If the error was due to a context cancelation/timeout of the context
		// we bail with the context error.
		if err == ctx.Err() {
			return err
		}

		logging.Log(
			w.Logger,
			"up-to-date snapshot of aggregate root %s[%s] cannot be be written at revision %d: %w",
			w.HandlerIdentity.Name,
			w.InstanceID,
			snapshotRev,
			err,
		)

		return nil
	}

	w.snapshotAge = 0

	logging.Log(
		w.Logger,
		"up-to-date snapshot of aggregate root %s[%s] written at revision %d: %w",
		w.HandlerIdentity.Name,
		w.InstanceID,
		snapshotRev,
	)

	return nil
}

// archiveSnapshots archives any existing snapshots of this instance.
//
// A failure to archive snapshots is not treated as an error; err is non-nil
// only if ctx is canceled. A failed attempt at archiving is never retried.
func (w *Worker) archiveSnapshots(ctx context.Context) error {
	if w.SnapshotWriter == nil {
		return nil
	}

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

	w.snapshotAge = 0

	return nil
}
