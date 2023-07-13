package fsm

import (
	"context"
)

// fsm is the internal state of a finite state machine.
type fsm struct {
	ctx            context.Context
	current, final State
	err            error
}

// execute runs s and applies the returned action to m.
func (m *fsm) execute(s State) {
	act := s(m.ctx)
	if act.apply == nil {
		panic("state must return a valid action")
	}
	act.apply(m)
}

// Start runs the state machine until it is stopped or an error occurs.
func Start(ctx context.Context, initial State, options ...Option) error {
	m := &fsm{
		ctx:     ctx,
		current: initial,
	}

	for _, opt := range options {
		opt.apply(m)
	}

	for m.current != nil {
		m.execute(m.current)

		if m.current == nil && m.final != nil {
			m.execute(m.final)
		}
	}

	return m.err
}

// Option is an option that changes the behavior of a state machine.
type Option struct {
	apply func(*fsm)
}

// WithFinalState is an option that sets the final state of a state machine.
//
// The final state is entered whenever the state machine stops. It may be used
// to prevent the state machine from stopping (by explicitly entering another
// state) and/or to perform cleanup actions.
func WithFinalState(s State) Option {
	return Option{func(m *fsm) {
		m.final = s
	}}
}
