package aggregate

import (
	"context"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
)

// DefaultCommandBuffer is the default number of commands to buffer in memory
// per aggregate instance.
const DefaultCommandBuffer = 100

// Supervisor manages the lifecycle of all workers for a specific aggregate.
type Supervisor struct {
	// Handler is the message handler that is managed by this supervisor.
	Handler dogma.AggregateMessageHandler

	// HandlerIdentity is the identity of the handler.
	HandlerIdentity configkit.Identity

	// StartWorker starts a worker for a specific instance.
	StartWorker func(
		ctx context.Context,
		id string,
		idle chan<- string,
		done func(error),
	) chan<- *Command

	// Commands is the channel on which commands are received before being
	// dispatched to instance-specific workers.
	//
	// If the channel is buffered, commands may be ignored if the supervisor
	// shuts down due to an error.
	Commands <-chan *Command

	// workerCommands is a map of aggregate instance IDs to channels on which
	// commands can be sent to the worker for that instance.
	workerCommands map[string]chan<- *Command

	// workerIdle is a channel that receives requests from workers to shut down
	// when they have entered an idle state.
	workerIdle chan string

	// workerShutdown is a channel that receives results from workers when their
	// Run() method returns.
	workerShutdown chan workerResult

	// currentState is the current state of the worker.
	currentState supervisorState
}

// workerResult is the result of a worker exiting.
type workerResult struct {
	InstanceID string
	Err        error
}

// supervisorState is a function that provides supervisor logic for a specific
// state.
//
// It returns the next state that the supervisor should transition to, or nil to
// indicate that the supervisor is done.
type supervisorState func(context.Context) (supervisorState, error)

// Run runs the supervisor until ctx is canceled or an error occurs.
func (s *Supervisor) Run(ctx context.Context) error {
	defer s.waitForAllWorkers()

	// Setup a context that is always canceled when the supervisor stops, so
	// that workers are also stopped.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.workerCommands = map[string]chan<- *Command{}
	s.workerIdle = make(chan string)
	s.workerShutdown = make(chan workerResult)
	s.currentState = s.stateWaitForCommand

	for {
		var err error
		s.currentState, err = s.currentState(ctx)
		if err != nil {
			return err
		}
	}
}

// stateWaitForCommand blocks until a command is available for dispatching.
func (s *Supervisor) stateWaitForCommand(ctx context.Context) (supervisorState, error) {
	select {
	case cmd := <-s.Commands:
		return s.stateDispatchCommand(cmd), nil

	case id := <-s.workerIdle:
		s.shutdownWorker(id)
		return s.currentState, nil

	case res := <-s.workerShutdown:
		return s.currentState, s.handleShutdown(res)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// stateDispatchCommand dispatches a command to the appropriate worker based on
// the aggregate instance ID.
//
// If the worker is not already running it is started.
func (s *Supervisor) stateDispatchCommand(cmd *Command) supervisorState {
	return func(ctx context.Context) (supervisorState, error) {
		id := s.Handler.RouteCommandToInstance(cmd.Parcel.Message)
		commands := s.startWorkerIfNotRunning(ctx, id)

		select {
		case commands <- cmd:
			return s.stateWaitForCommand, nil

		case idleID := <-s.workerIdle:
			if idleID != id {
				// Only shut the worker down if it's NOT the one we're trying to
				// send a command to.
				s.shutdownWorker(idleID)
			}
			return s.currentState, nil

		case res := <-s.workerShutdown:
			err := s.handleShutdown(res)
			if err != nil {
				cmd.Nack(errShutdown)
				return nil, err
			}
			return s.currentState, err

		case <-ctx.Done():
			cmd.Nack(errShutdown)
			return nil, ctx.Err()

		case <-cmd.Context.Done():
			cmd.Nack(cmd.Context.Err())
			return s.stateWaitForCommand, nil
		}
	}
}

// startWorkerIfNotRunning starts a worker for the given aggregate instance if
// it is not already running.
//
// It returns a channel on which commands can be sent to the worker.
func (s *Supervisor) startWorkerIfNotRunning(
	ctx context.Context,
	id string,
) chan<- *Command {
	if commands, ok := s.workerCommands[id]; ok {
		return commands
	}

	commands := s.StartWorker(
		ctx,
		id,
		s.workerIdle,
		func(err error) {
			s.workerShutdown <- workerResult{id, err}
		},
	)
	s.workerCommands[id] = commands

	return commands
}

// waitForAllWorkers blocks until all workers have stopped running.
func (s *Supervisor) waitForAllWorkers() {
	for len(s.workerCommands) != 0 {
		s.handleShutdown(<-s.workerShutdown)
	}
}

// shutdownWorker signals to a worker that it should shut down.
func (s *Supervisor) shutdownWorker(id string) {
	commands := s.workerCommands[id]
	if commands == nil {
		// The worker has already been instructed to shutdown.
		return
	}

	// Close the channel to signal to the worker that it should shut down.
	close(commands)

	// Don't remove the worker from the map yet, because it has not exited.
	// Instead we use a nil channel so that writes always block.
	//
	// This prevents the supervisor from starting another worker for the same
	// instance before this one has actually shutdown.
	s.workerCommands[id] = nil
}

// handleShutdown handles a result from a worker's Run() method.
func (s *Supervisor) handleShutdown(res workerResult) error {
	delete(s.workerCommands, res.InstanceID)
	return res.Err
}
