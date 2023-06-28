package fsm

import (
	"sync/atomic"
)

// Future represents a future value of type T.
type Future[T any] struct {
	r atomic.Pointer[chan struct{}]
	v atomic.Pointer[T]
}

// Ready returns a channel that is closed when the value is ready.
func (f *Future[T]) Ready() <-chan struct{} {
	return f.ready()
}

// Get returns the value. It panics if the value is not ready.
func (f *Future[T]) Get() T {
	if v := f.v.Load(); v != nil {
		return *v
	}
	panic("future is not resolved")
}

// Set resolves the future with the given value.
func (f *Future[T]) Set(v T) {
	if !f.v.CompareAndSwap(nil, &v) {
		panic("future is already resolved")
	}

	r := f.ready()
	close(r)
}

func (f *Future[T]) ready() chan struct{} {
	if ch := f.r.Load(); ch != nil {
		return *ch
	}

	ch := make(chan struct{})
	if f.r.CompareAndSwap(nil, &ch) {
		return ch
	}

	return *f.r.Load()
}

// FailableFuture represents a future value of type T, or an error indicating
// that the value can not be resolved.
type FailableFuture[T any] struct {
	f Future[maybe[T]]
}

// Ready returns a channel that is closed when the value is ready.
func (f *FailableFuture[T]) Ready() <-chan struct{} {
	return f.f.Ready()
}

// Get returns the value. It panics if the value is not ready.
func (f *FailableFuture[T]) Get() (T, error) {
	v := f.f.Get()
	return v.Value, v.Err
}

// Set resolves the future with the given value.
func (f *FailableFuture[T]) Set(v T) {
	f.f.Set(maybe[T]{Value: v})
}

// Err resolves the future within the given error.
func (f *FailableFuture[T]) Err(err error) {
	f.f.Set(maybe[T]{Err: err})
}

type maybe[T any] struct {
	Value T
	Err   error
}
