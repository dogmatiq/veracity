package fsm

import "context"

type (
	// State is a function that implements the logic for a single state.
	State func(context.Context) Action

	// State1 is a state that accept a single argument.
	State1[T1 any] func(context.Context, T1) Action

	// State2 is a state that accepts two arguments.
	State2[T1, T2 any] func(context.Context, T1, T2) Action
)

// Binding returns actions that can be performed by the state machine within the
// context of some additional arguments.
type Binding[S any] struct {
	EnterState func(S) Action
}

// With binds a state to an argument.
func With[T1 any](v1 T1) Binding[State1[T1]] {
	return Binding[State1[T1]]{
		func(s State1[T1]) Action {
			return EnterState(
				func(ctx context.Context) Action {
					return s(ctx, v1)
				},
			)
		},
	}
}

// With2 binds a state to 2 arguments.
func With2[T1, T2 any](v1 T1, v2 T2) Binding[State2[T1, T2]] {
	return Binding[State2[T1, T2]]{
		func(s State2[T1, T2]) Action {
			return EnterState(
				func(ctx context.Context) Action {
					return s(ctx, v1, v2)
				},
			)
		},
	}
}
