package future

import (
	"context"
	"sync/atomic"
)

// New returns a future that can be fulfilled with a value of type T,
// and its associated resolver.
func New[T any]() (Future[T], Resolver[T]) {
	f := &state[T]{
		Ready: make(chan struct{}),
	}

	return Future[T]{f}, Resolver[T]{f}
}

// Future represents a future value of type T.
type Future[T any] struct {
	f *state[T]
}

// Ready returns a channel that is closed when the value is ready.
func (f Future[T]) Ready() <-chan struct{} {
	return f.f.Ready
}

// Get returns the value. It panics if the value is not ready.
func (f Future[T]) Get() T {
	if v := f.f.Value.Load(); v != nil {
		return *v
	}
	panic("future value is not ready")
}

// Wait blocks until the value is ready, then returns it.
func (f Future[T]) Wait(ctx context.Context) (T, error) {
	if v := f.f.Value.Load(); v != nil {
		return *v, nil
	}

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-f.f.Ready:
		v := f.f.Value.Load()
		return *v, nil
	}
}

// Resolver is used to provide a result value to a [Failable].
type Resolver[T any] struct {
	f *state[T]
}

// Set resolves the future with the given value.
func (r Resolver[T]) Set(v T) {
	if !r.f.Value.CompareAndSwap(nil, &v) {
		panic("future has already been resolved")
	}
	close(r.f.Ready)
}

// IsResolved returns true if the future has been resolved, either successfully
// or with an error.
func (r Resolver[T]) IsResolved() bool {
	return r.f.Value.Load() != nil
}

type state[T any] struct {
	Ready chan struct{}
	Value atomic.Pointer[T]
}
