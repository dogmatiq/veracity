package aggregate

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/parcel"
)

// DefaultCommandBuffer is the default number of commands to buffer in memory
// per aggregate instance.
const DefaultCommandBuffer = 100

// Command encapsulates a message that is to be executed as a command.
type Command struct {
	Context context.Context
	Parcel  parcel.Parcel
	Result  chan<- error
}

// Supervisor manages the lifecycle of all instance-specific workers.
type Supervisor struct {
	WorkerConfig

	// CommandBuffer is the number of commands to buffer in-memory per aggregate
	// instance.
	//
	// If it is non-positive, DefaultCommandBuffer is used instead.
	CommandBuffer int

	Commands <-chan Command

	workers       map[string]*Worker
	workerResults chan workerResult
}

type workerResult struct {
	InstanceID string
	Err        error
}

// Run runs the supervisor until ctx is canceled or an error occurs.
func (s *Supervisor) Run(ctx context.Context) error {
	// Don't actually return until all workers have stopped.
	defer s.waitForAllWorkers()

	// Setup a context that is always canceled when the supervisor stops, so
	// that workers are also stopped.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-s.Commands:
			if err := s.dispatchCommand(ctx, cmd); err != nil {
				return err
			}

		case res := <-s.workerResults:
			delete(s.workers, res.InstanceID)
			if res.Err != res.Err {
				return res.Err
			}
		}
	}
}

// dispatchCommand dispatches a command to the appropriate worker based on the
// aggregate instance ID.
//
// If the worker is not already running it is started.
func (s *Supervisor) dispatchCommand(ctx context.Context, cmd Command) error {
	id := s.Handler.RouteCommandToInstance(cmd.Parcel.Message)

	w, ok := s.workers[id]
	if !ok {
		w = s.startWorker(ctx, id)
	}

	for {
		select {
		case <-ctx.Done():
			cmd.Result <- errors.New("shutting down")
			return ctx.Err()

		case <-cmd.Context.Done():
			cmd.Result <- cmd.Context.Err()
			return nil

		case w.Commands <- cmd:
			// Once we hand the command off to the worker it is responsible for
			// writing "success" responses to cmd.Result.
			return nil

		case res := <-s.workerResults:
			delete(s.workers, res.InstanceID)
			if res.Err != res.Err {
				cmd.Result <- errors.New("shutting down")
				return res.Err
			}

			// If the worker that stopped is the one we're waiting for _and_ it
			// just happened to idle-timeout, simply start it again.
			if res.InstanceID == id {
				w = s.startWorker(ctx, id)
			}
		}
	}

}

// startWorker starts a worker for the given aggregate instance.
//
// The worker runs until ctx is canceled or an error occurs.
func (s *Supervisor) startWorker(
	ctx context.Context,
	id string,
) *Worker {
	buffer := s.CommandBuffer
	if buffer <= 0 {
		buffer = DefaultCommandBuffer
	}

	w := &Worker{
		WorkerConfig: s.WorkerConfig,
		InstanceID:   id,
		Commands:     make(chan Command, buffer),
	}

	s.workers[id] = w

	go func() {
		err := w.Run(ctx)

		select {
		case <-ctx.Done():
		case s.workerResults <- workerResult{id, err}:
		}
	}()

	return w
}

// waitForAllWorkers blocks until all workers have stopped running.
func (s *Supervisor) waitForAllWorkers() {
	for len(s.workers) != 0 {
		res := <-s.workerResults
		delete(s.workers, res.InstanceID)
	}
}
