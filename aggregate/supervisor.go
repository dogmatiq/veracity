package aggregate

import (
	"context"

	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dodeca/logging"
)

// DefaultCommandBuffer is the default number of commands to buffer in memory
// per aggregate instance.
const DefaultCommandBuffer = 100

// Supervisor manages the lifecycle of all workers for a specific aggregate.
type Supervisor struct {
	// WorkerConfig is the configuration used for each worker started by the
	// supervisor.
	WorkerConfig

	// Commands is the channel on which commands are received before being
	// dispatched to instance-specific workers.
	//
	// If the channel is buffered, commands may be ignored if the supervisor
	// shuts down due to an error.
	Commands <-chan *Command

	// CommandBuffer is the number of commands to buffer in-memory per aggregate
	// instance.
	//
	// If it is non-positive, DefaultCommandBuffer is used instead.
	CommandBuffer int

	// workerCommands is a map of aggregate instance IDs to channels on which
	// commands can be sent to the worker for that instance.
	workerCommands map[string]chan *Command

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

	s.workerCommands = map[string]chan *Command{}
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
		id := s.Handler.RouteCommandToInstance(cmd.Parcel.Message)

		if s.Observer != nil {
			s.Observer.OnCommandReceived(
				s.HandlerIdentity,
				id,
				cmd,
			)
		}

		logging.Log(
			s.Logger,
			"command %s[%s] received for aggregate %s[%s]",
			message.TypeOf(cmd.Parcel.Message),
			cmd.Parcel.Envelope.GetMessageId(),
			s.HandlerIdentity.Name,
			id,
		)

		return s.stateDispatchCommand(id, cmd), nil

	case id := <-s.workerIdle:
		if s.Observer != nil {
			s.Observer.OnWorkerIdle(
				s.HandlerIdentity,
				id,
			)
		}

		s.acknowledgeIdle(id)

		return s.currentState, nil

	case res := <-s.workerShutdown:
		return s.currentState, s.handleShutdown(res)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// stateDispatchCommand dispatches a command to the appropriate worker based on the
// aggregate instance ID.
//
// If the worker is not already running it is started.
func (s *Supervisor) stateDispatchCommand(id string, cmd *Command) supervisorState {
	return func(ctx context.Context) (supervisorState, error) {
		commands := s.startWorkerIfNotRunning(ctx, id)

		select {
		case commands <- cmd:
			if s.Observer != nil {
				s.Observer.OnCommandDispatched(
					s.HandlerIdentity,
					id,
					cmd,
				)
			}

			logging.Log(
				s.Logger,
				"command %s[%s] enqueued for aggregate %s[%s] worker",
				message.TypeOf(cmd.Parcel.Message),
				cmd.Parcel.Envelope.GetMessageId(),
				s.HandlerIdentity.Name,
				id,
			)

			return s.stateWaitForCommand, nil

		case idleID := <-s.workerIdle:
			if s.Observer != nil {
				s.Observer.OnWorkerIdle(
					s.HandlerIdentity,
					idleID,
				)
			}

			// Only shut the worker down if it's NOT the one we're trying to
			// send a command to.
			if idleID == id {
				logging.Log(
					s.Logger,
					"aggregate %s[%s] worker is idle but is about to be sent another command",
					s.HandlerIdentity.Name,
					id,
				)
			} else {
				s.acknowledgeIdle(idleID)
			}

			return s.currentState, nil

		case res := <-s.workerShutdown:
			err := s.handleShutdown(res)
			if err != nil {
				s.nackCommand(id, cmd, errShutdown, err)
				return nil, err
			}

			return s.currentState, err

		case <-ctx.Done():
			s.nackCommand(id, cmd, errShutdown, ctx.Err())
			return nil, ctx.Err()

		case <-cmd.Context.Done():
			err := cmd.Context.Err()
			s.nackCommand(id, cmd, err, err)
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
) chan *Command {
	if commands, ok := s.workerCommands[id]; ok {
		return commands
	}

	buffer := s.CommandBuffer
	if buffer <= 0 {
		buffer = DefaultCommandBuffer
	}

	commands := make(chan *Command, buffer)
	s.workerCommands[id] = commands

	go func() {
		w := &Worker{
			WorkerConfig: s.WorkerConfig,
			InstanceID:   id,
			Commands:     commands,
			Idle:         s.workerIdle,
		}

		if s.Observer != nil {
			s.Observer.OnWorkerStart(
				s.HandlerIdentity,
				id,
			)
		}

		logging.Log(
			s.Logger,
			"aggregate %s[%s] worker has started",
			s.HandlerIdentity.Name,
			id,
		)

		err := w.Run(ctx)
		s.workerShutdown <- workerResult{id, err}
	}()

	return commands
}

// waitForAllWorkers blocks until all workers have stopped running.
func (s *Supervisor) waitForAllWorkers() {
	for len(s.workerCommands) != 0 {
		s.handleShutdown(<-s.workerShutdown)
	}
}

// acknowledgeIdle signals to a worker that it should shut down by closing the
// worker's command channel.
func (s *Supervisor) acknowledgeIdle(id string) {
	commands := s.workerCommands[id]
	if commands == nil {
		// The worker has already been instructed to shutdown.
		return
	}

	logging.Log(
		s.Logger,
		"aggregate %s[%s] worker is idle and will be shutdown",
		s.HandlerIdentity.Name,
		id,
	)

	if s.Observer != nil {
		s.Observer.OnWorkerIdleAck(
			s.HandlerIdentity,
			id,
		)
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
	if s.Observer != nil {
		s.Observer.OnWorkerShutdown(
			s.HandlerIdentity,
			res.InstanceID,
			res.Err,
		)
	}

	if res.Err == nil {
		logging.Log(
			s.Logger,
			"aggregate %s[%s] worker has shutdown",
			s.HandlerIdentity.Name,
			res.InstanceID,
		)
	} else {
		logging.Log(
			s.Logger,
			"aggregate %s[%s] worker has shutdown: %s",
			s.HandlerIdentity.Name,
			res.InstanceID,
			res.Err,
		)
	}

	delete(s.workerCommands, res.InstanceID)

	return res.Err
}

// nackCommand sends a NACK to the given command.
func (s *Supervisor) nackCommand(
	id string,
	cmd *Command,
	reported, cause error,
) {
	if s.Observer != nil {
		s.Observer.OnCommandNack(
			s.HandlerIdentity,
			id,
			cmd,
			reported,
			cause,
		)
	}

	cmd.Nack(reported)
}
