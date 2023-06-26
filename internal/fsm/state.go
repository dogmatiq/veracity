package fsm

import "context"

// State is a function that implements the logic for a single state.
type State func(context.Context) (Action, error)

// Start runs the state machine until it is stopped or an error occurs.
func Start(ctx context.Context, initial State) error {
	m := &fsm{initial, nil}

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
	current, previous State
}

// Action describes the action taken by a state.
type Action func(*fsm)

// Stay is an action that stays in the current state.
func Stay() Action {
	return func(*fsm) {}
}

// Stop is an action that stops the state machine.
func Stop() Action {
	return func(m *fsm) {
		m.current = nil
	}
}

// Back is an action that transitions to the previous state.
func Back() Action {
	return func(m *fsm) {
		m.current, m.previous = m.previous, m.current
	}
}

// EnterState returns an action that transitions to a new state.
func EnterState(s State) Action {
	return func(m *fsm) {
		m.previous = m.current
		m.current = s
	}
}

// EnterStateWith1Arg returns an action that transitions to a state that requires 1
// parameter.
func EnterStateWith1Arg[T1 any](
	s func(context.Context, T1) (Action, error),
	v1 T1,
) Action {
	return EnterState(func(ctx context.Context) (Action, error) {
		return s(ctx, v1)
	})
}

// EnterStateWith2Args returns an action that transitions to a state that
// requires 2 parameters.
func EnterStateWith2Args[T1, T2 any](
	s func(context.Context, T1, T2) (Action, error),
	v1 T1,
	v2 T2,
) Action {
	return EnterState(func(ctx context.Context) (Action, error) {
		return s(ctx, v1, v2)
	})
}
