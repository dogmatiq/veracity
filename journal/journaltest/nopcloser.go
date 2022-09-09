package journaltest

import (
	"github.com/dogmatiq/veracity/journal"
)

// NopCloser returns a Journal with a no-op Close() method wrapping the provided
// Journal j.
func NopCloser[R any](j journal.Journal[R]) journal.Journal[R] {
	return nopCloser[R]{j}
}

type nopCloser[R any] struct {
	journal.Journal[R]
}

func (j nopCloser[R]) Close() error {
	return nil
}
