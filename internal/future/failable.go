package future

import (
	"context"
)

// NewFailable returns a future that can be fulfilled with a value of type T or
// an error, and its associated resolver.
func NewFailable[T any]() (Failable[T], FailableResolver[T]) {
	f, r := New[failable[T]]()
	return Failable[T]{f}, FailableResolver[T]{r}
}

// Failable represents a future value of type T, or an error indicating that the
// value can not be computed.
type Failable[T any] struct {
	fut Future[failable[T]]
}

// Ready returns a channel that is closed when the value is ready.
func (f Failable[T]) Ready() <-chan struct{} {
	return f.fut.Ready()
}

// Get returns the value. It panics if the value is not ready.
func (f Failable[T]) Get() (T, error) {
	v := f.fut.Get()
	return v.Value, v.Err
}

// Wait blocks until the value is ready, then returns it.
func (f Failable[T]) Wait(ctx context.Context) (T, error) {
	v, err := f.fut.Wait(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return v.Value, v.Err
}

// FailableResolver is used to provide a result value to a [Failable].
type FailableResolver[T any] struct {
	res Resolver[failable[T]]
}

// Set resolves the future with the given value.
func (r FailableResolver[T]) Set(v T) {
	r.res.Set(failable[T]{Value: v})
}

// Err resolves the future within the given error.
func (r FailableResolver[T]) Err(err error) {
	r.res.Set(failable[T]{Err: err})
}

// IsResolved returns true if the future has been resolved, either successfully
// or with an error.
func (r FailableResolver[T]) IsResolved() bool {
	return r.res.IsResolved()
}

type failable[T any] struct {
	Value T
	Err   error
}
