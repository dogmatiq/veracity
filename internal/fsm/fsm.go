package fsm

import "context"

type (
	// State is a function that implements the logic for a single state.
	State func(context.Context) (Action, error)

	// State1 is a state that accept a single argument.
	State1[T1 any] func(context.Context, T1) (Action, error)

	// State2 is a state that accepts two arguments.
	State2[T1, T2 any] func(context.Context, T1, T2) (Action, error)
)

// Start runs the state machine until it is stopped or an error occurs.
func Start(ctx context.Context, initial State) error {
	m := &fsm{initial, nil}

	for m.current != nil {
		act, err := m.current(ctx)
		if err != nil {
			return err
		}

		if act == nil {
			panic("action must not be nil")
		}

		act(m)
	}

	return nil
}

// Action describes the action taken by a state.
type Action func(*fsm)

// Stop is an action that stops the state machine.
func Stop() (Action, error) {
	return func(m *fsm) {
		m.current, m.previous = nil, m.current
	}, nil
}

// StayInCurrentState is an action that stays in the current state.
func StayInCurrentState() (Action, error) {
	return func(*fsm) {}, nil
}

// ReturnToPreviousState is an action that transitions to the previous state.
func ReturnToPreviousState() (Action, error) {
	return func(m *fsm) {
		m.current, m.previous = m.previous, m.current
	}, nil
}

// Enter returns an action that transitions to a new state.
func Enter(s State) (Action, error) {
	if s == nil {
		panic("state must not be nil")
	}

	return func(m *fsm) {
		m.previous = m.current
		m.current = s
	}, nil
}

// With returns a binding that enters a state with a single argument.
func With[T1 any](v1 T1) Binding[State1[T1]] {
	return Binding[State1[T1]]{
		func(s State1[T1]) (Action, error) {
			return Enter(
				func(ctx context.Context) (Action, error) {
					return s(ctx, v1)
				},
			)
		},
	}
}

// With2 returns a binding that enters a state with two arguments.
func With2[T1, T2 any](v1 T1, v2 T2) Binding[State2[T1, T2]] {
	return Binding[State2[T1, T2]]{
		func(s State2[T1, T2]) (Action, error) {
			return Enter(
				func(ctx context.Context) (Action, error) {
					return s(ctx, v1, v2)
				},
			)
		},
	}
}

// Binding returns actions that can be performed by the state machine within the
// context of some additional arguments.
type Binding[S any] struct {
	Enter func(S) (Action, error)
}

// fsm is the internal state of a finite state machine.
type fsm struct {
	current, previous State
}
