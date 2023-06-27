package signal

import "sync"

// Lossy is a signal that sends a value to a set of buffered watcher channels
// when it is notified.
//
// Any watchers that have a prior unhandled values are ignored.
type Lossy struct {
	m        sync.RWMutex
	watchers map[chan<- struct{}]struct{}
}

var _ Signal = (*Lossy)(nil)

// Watch registers a buffered watcher channel to receive a value when the signal
// is next notified.
func (s *Lossy) Watch(watcher chan<- struct{}) CancelFunc {
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
func (s *Lossy) Notify() {
	s.m.RLock()
	defer s.m.RUnlock()

	for ch := range s.watchers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
