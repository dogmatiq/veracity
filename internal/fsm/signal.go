package fsm

import (
	"sync"
	"sync/atomic"
)

// CancelFunc is a function that cancels a watcher from receiving signals.
type CancelFunc func()

// Latch is signal that closes a set of "watcher channels" when it is notified.
//
// Once notified, the signal "latches on", such that any future watcher channels
// are closed immediately.
type Latch struct {
	done     atomic.Bool
	m        sync.Mutex
	watchers map[chan<- struct{}]struct{}
}

// Watch registers a watcher channel to be closed when the signal is notified.
// If s has already been notified, the watcher is closed immediately.
func (s *Latch) Watch(watcher chan<- struct{}) CancelFunc {
	if s.done.Load() {
		close(watcher)
		return func() {}
	}

	s.m.Lock()
	defer s.m.Unlock()

	if s.done.Load() {
		close(watcher)
		return func() {}
	}

	if s.watchers == nil {
		s.watchers = map[chan<- struct{}]struct{}{}
	}
	s.watchers[watcher] = struct{}{}

	return func() {
		s.m.Lock()
		defer s.m.Unlock()
		delete(s.watchers, watcher)
	}
}

// Notify closes all watcher channels.
//
// Any future watchers will be closed immediately.
func (s *Latch) Notify() {
	if s.done.CompareAndSwap(false, true) {
		s.m.Lock()
		defer s.m.Unlock()

		for ch := range s.watchers {
			close(ch)
		}

		s.watchers = nil
	}
}

// Condition is a signal that sends a value to a set of buffered watcher channels
// when it is notified.
//
// Any watchers that have a prior unhandled values are ignored.
type Condition struct {
	m        sync.RWMutex
	watchers map[chan<- struct{}]struct{}
}

// Watch registers a buffered watcher channel to receive a value when the signal
// is next notified.
func (s *Condition) Watch(watcher chan<- struct{}) CancelFunc {
	if cap(watcher) == 0 {
		panic("channel must be buffered")
	}

	s.m.Lock()
	defer s.m.Unlock()

	if s.watchers == nil {
		s.watchers = map[chan<- struct{}]struct{}{}
	}

	s.watchers[watcher] = struct{}{}

	return func() {
		s.m.Lock()
		defer s.m.Unlock()
		delete(s.watchers, watcher)
	}
}

// Notify sends a value to each watcher that do not have a prior unhandled
// notification.
func (s *Condition) Notify() {
	s.m.RLock()
	defer s.m.RUnlock()

	for ch := range s.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
