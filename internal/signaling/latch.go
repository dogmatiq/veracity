package signaling

import (
	"sync"
	"sync/atomic"
)

// Latch is a signal that indicates some permanent condition has been met.
type Latch struct {
	init    sync.Once
	sig     chan struct{}
	latched atomic.Bool
}

// Signaled returns a channel that is readable once the latch has been set.
func (l *Latch) Signaled() <-chan struct{} {
	return l.signal()
}

// Signal sets the latch.
func (l *Latch) Signal() {
	if l.latched.CompareAndSwap(false, true) {
		close(l.signal())
	}
}

func (l *Latch) signal() chan struct{} {
	l.init.Do(func() {
		l.sig = make(chan struct{})
	})
	return l.sig
}
