package aggregate

import (
	"context"
	"errors"
	"sync"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
)

// DefaultRequestBuffer is the default number of command requests to buffer in
// memory per aggregate instance.
const DefaultRequestBuffer = 100

// EventAppender is an interface for appending event messages to a stream.
type EventAppender interface {
	Append(ctx context.Context, envelopes ...*envelopespec.Envelope) error
}

// CommandExecutor coordinates execution of commands by a specific aggregate
// message handler.
type CommandExecutor struct {
	// HandlerIdentity is the identity of the handler used to handle commands.
	HandlerIdentity *envelopespec.Identity

	// Handler is the handler used to handle commands.
	Handler dogma.AggregateMessageHandler

	// Packer is used to pack envelopes that contain the domain events that are
	// recorded by the handler.
	Packer *envelope.Packer

	// JournalStore is the store contain the aggregate instances' journals.
	JournalStore journal.Store[*JournalRecord]

	// EventAppender is used to append events to the global event stream.
	EventAppender EventAppender

	// RequestBuffer is the number of requests to buffer in memory per aggregate
	// instance.
	//
	// If it is non-positive, DefaultRequestBuffer is used instead.
	RequestBuffer int

	// Logger is the target for messages about the execution of commands and
	// management of aggregate state.
	Logger *zap.Logger

	instances map[string]chan<- request

	once     sync.Once
	abort    chan struct{}
	requests chan request
	unloads  chan unload
}

// request encapsulates a request to execute a command.
type request struct {
	InstanceID      string
	CommandEnvelope *envelopespec.Envelope
	Done            chan<- struct{}
}

// unload contains information about an aggregate instance that has been
// unloaded.
type unload struct {
	InstanceID string
	Error      error
}

// ExecuteCommand executes a command against the given aggregate instance.
func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	id string,
	env *envelopespec.Envelope,
) error {
	e.init()

	done := make(chan struct{})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.abort:
		return errors.New("executor has stopped")
	case e.requests <- request{id, env, done}:
		break // coverage
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.abort:
		return errors.New("executor has stopped")
	case <-done:
		return nil
	}
}

// Run starts the command executor.
//
// It runs until ctx is canceled or an error occurs.
func (e *CommandExecutor) Run(ctx context.Context) error {
	e.init()

	// Don't let Run() exit until we've cleaned up all of the goroutines we
	// started.
	defer e.waitForUnloads()

	// Setup a context that is always canceled when the executor stops so that
	// instances are also stopped.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Abort any pending calls to ExecuteCommand() that will not be serviced.
	defer close(e.abort)

	return fsm.Run(ctx, e.stateAwait)
}

// stateAwait waits for a command request to be received.
func (e *CommandExecutor) stateAwait(ctx context.Context) (fsm.Action, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-e.requests:
		return fsm.TransitionWith(e.stateDispatch, r), nil
	case u := <-e.unloads:
		return fsm.StayInCurrentState, e.handleUnload(u)
	}
}

// stateDispatch dispatches a command request to the appropriate instance.
func (e *CommandExecutor) stateDispatch(
	ctx context.Context,
	req request,
) (fsm.Action, error) {
	requests := e.loadInstance(ctx, req.InstanceID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case requests <- req:
		return fsm.Transition(e.stateAwait), nil
	}
}

// init initializes the executor's channels.
func (e *CommandExecutor) init() {
	e.once.Do(func() {
		e.abort = make(chan struct{})
		e.requests = make(chan request)
		e.unloads = make(chan unload)
	})
}

// loadInstance starts loading an aggregate instance into memory (if it is not
// already loaded).
//
// It returns the channel used to dispatch requests to the instance.
func (e *CommandExecutor) loadInstance(
	ctx context.Context,
	id string,
) chan<- request {
	if requests, ok := e.instances[id]; ok {
		return requests
	}

	cap := e.RequestBuffer
	if cap <= 0 {
		cap = DefaultRequestBuffer
	}
	requests := make(chan request, cap)

	if e.instances == nil {
		e.instances = map[string]chan<- request{}
	}
	e.instances[id] = requests

	go e.runInstance(ctx, id, requests)

	return requests
}

// runInstance loads an runs an aggregate instance.
func (e *CommandExecutor) runInstance(
	ctx context.Context,
	id string,
	requests <-chan request,
) {
	e.Logger.Debug(
		"loading aggregate instance",
		zap.String("instance_id", id),
	)

	err := func() error {
		j, err := e.JournalStore.Open(
			ctx,
			"aggregate",
			e.HandlerIdentity.Key,
			id,
		)
		if err != nil {
			return err
		}
		defer j.Close()

		inst := &instance{
			HandlerIdentity: e.HandlerIdentity,
			InstanceID:      id,
			Handler:         e.Handler,
			Requests:        requests,
			Journal:         j,
			Packer:          e.Packer,
			EventAppender:   e.EventAppender,
			Logger: e.Logger.With(
				zap.String("instance_id", id),
			),
		}

		return inst.Run(ctx)
	}()

	e.unloads <- unload{id, err}
}

// waitForUnloads blocks until all instances have been unloaded.
//
// It assumes the instances have already be signalled to stop.
func (e *CommandExecutor) waitForUnloads() {
	for len(e.instances) != 0 {
		e.handleUnload(<-e.unloads)
	}
}

// handleUnload handles an instance being unloaded.
func (e *CommandExecutor) handleUnload(u unload) error {
	delete(e.instances, u.InstanceID)

	if u.Error == nil || u.Error == context.Canceled {
		e.Logger.Debug(
			"aggregate instance unloaded",
			zap.String("instance_id", u.InstanceID),
		)
	} else {
		e.Logger.Error(
			"aggregate instance unloaded",
			zap.String("instance_id", u.InstanceID),
			zap.Error(u.Error),
		)
	}

	return u.Error
}
