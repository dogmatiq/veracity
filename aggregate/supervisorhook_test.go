package aggregate

// SetIdleTestHook registers a hook to be called when the supervisor observes
// a worker enter the idle state.
func (s *Supervisor) SetIdleTestHook(hook func(id string)) {
	s.idleTestHook = hook
}

// SetShutdownTestHook registers a hook to be called when the supervisor
// observes a worker's Run() method return.
func (s *Supervisor) SetShutdownTestHook(hook func(id string, err error)) {
	s.shutdownTestHook = hook
}
