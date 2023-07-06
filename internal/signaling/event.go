package signaling

import (
	"sync"
)

// Event is a signal that indicates some event has occurred.
type Event struct {
	init sync.Once
	sig  chan struct{}
}

// Signaled returns a channel that is readable if the event has occurred since
// the last successful read.
func (e *Event) Signaled() <-chan struct{} {
	return e.signal()
}

// Signal signals occurance of the event.
func (e *Event) Signal() {
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
