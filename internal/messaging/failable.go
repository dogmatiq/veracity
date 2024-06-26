package messaging

// Failable encapsulates a value of type T, or an error indicating that the
// value cannot be obtained.
type Failable[T any] struct {
	value T
	err   error
}

// Get returns the value, or an error if the value cannot be obtained.
func (f Failable[T]) Get() (T, error) {
	return f.value, f.err
}
