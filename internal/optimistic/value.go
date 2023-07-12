package optimistic

import (
	"sync/atomic"
)

// Value is a value protected by optimistic concurrency control.
type Value[T any] struct {
	p atomic.Pointer[T]
}

// Load returns the current value.
func (v *Value[T]) Load() T {
	if p := v.p.Load(); p != nil {
		return *p
	}

	var zero T
	return zero
}

// Apply calls fn with the current value.
//
// If fn returns true, the value is replaced with the returned value. If the
// value has been replaced by another goroutine since the call to fn, the
// process is retried until it succeeds or fn returns false.
func (v *Value[T]) Apply(
	fn func(T) (T, bool),
) (T, bool) {
	var (
		before, after T
		modify        bool
	)

	for {
		p := v.p.Load()

		if p != nil {
			before = *p
		}

		after, modify = fn(before)
		if !modify {
			return before, false
		}

		if v.p.CompareAndSwap(p, &after) {
			return after, true
		}
	}
}
