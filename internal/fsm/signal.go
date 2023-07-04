package fsm

import (
	"sync"
	"sync/atomic"
)

// Event is a signal that indicates some event has occurred.
type Event struct {
	init sync.Once
	sig  chan struct{}
}

// Occurred returns a channel that is readable if the event has occurred since
// the last successful read.
func (e *Event) Occurred() <-chan struct{} {
	return e.signal()
}

// Notify signals that the event has occurred.
func (e *Event) Notify() {
	select {
	case e.signal() <- struct{}{}:
	default:
	}
}

func (e *Event) signal() chan struct{} {
	e.init.Do(func() {
		e.sig = make(chan struct{}, 1)
	})
	return e.sig
}

// Latch is a signal that indicates some permanent condition has been met.
type Latch struct {
	init    sync.Once
	sig     chan struct{}
	latched atomic.Bool
}

// Latched returns a channel that is readable once the latch has been set.
func (l *Latch) Latched() <-chan struct{} {
	return l.signal()
}

// Latch sets the latch.
func (l *Latch) Latch() {
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
