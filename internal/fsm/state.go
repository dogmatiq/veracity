package fsm

import "context"

// State is a function that implements the logic for a single state.
type State func(context.Context) (Action, error)

// Run runs the state machine until it is stopped or an error occurs.
func Run(ctx context.Context, initial State) error {
	m := &fsm{initial}

	for m.current != nil {
		act, err := m.current(ctx)
		if err != nil {
			return err
		}

		act(m)
	}

	return nil
}

type fsm struct {
	current State
}

// Action describes the action taken by a state.
type Action func(*fsm)

// StayInCurrentState is an action that stays in the current state.
func StayInCurrentState(*fsm) {
}

// Stop is an action that stops the state machine.
func Stop(m *fsm) {
	m.current = nil
}

// Transition returns an action that transitions to a new state.
func Transition(s State) Action {
	return func(m *fsm) {
		m.current = s
	}
}

// TransitionWith returns an action that transitions to a state that requires 1
// parameter.
func TransitionWith[T any](
	s func(context.Context, T) (Action, error),
	v T,
) Action {
	return func(m *fsm) {
		m.current = func(ctx context.Context) (Action, error) {
			return s(ctx, v)
		}
	}
}
