package test

import "sync"

// FailOnce returns a function that returns the given error once, and then
// returns nil on subsequent calls.
func FailOnce(err error) func() error {
	var once sync.Once

	return func() error {
		var e error
		once.Do(func() {
			e = err
		})
		return e
	}
}
