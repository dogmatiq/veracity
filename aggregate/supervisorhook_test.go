package aggregate

// SetIdleTestHook registers a hook to be called  when the supervisor observes
// a worker enter the idle state.
func (s *Supervisor) SetIdleTestHook(hook func()) {
	s.idleTestHook = hook
}
