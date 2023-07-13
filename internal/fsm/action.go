package fsm

// Action describes the action taken by a state.
type Action struct {
	apply func(*fsm)
}

// Stop is an action that stops the state machine.
func Stop() Action {
	return Action{func(m *fsm) {
		m.current = nil
		m.err = m.ctx.Err()
	}}
}

// Fail is an action that stops the state machine with an error.
func Fail(err error) Action {
	return Action{func(m *fsm) {
		m.current = nil
		m.err = err
	}}
}

// StayInCurrentState is an action that stays in the current state.
func StayInCurrentState() Action {
	return Action{func(*fsm) {}}
}

// EnterState returns an action that transitions to a new state.
func EnterState(next State) Action {
	if next == nil {
		panic("state must not be nil")
	}

	return Action{func(m *fsm) {
		m.current = next
	}}
}
