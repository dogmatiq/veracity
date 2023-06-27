package signal

import (
	"sync"
	"sync/atomic"
)

// Latch is signal that closes a set of "watcher channels" when it is notified.
//
// Once notified, the signal "latches on", such that any future watcher channels
// are closed immediately.
type Latch struct {
	done     atomic.Bool
	m        sync.Mutex
	watchers map[chan<- struct{}]struct{}
}

var _ Signal = (*Latch)(nil)

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
