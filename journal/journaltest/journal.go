package journaltest

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/dogmatiq/veracity/journal"
)

// JournalStub is a test implementation of the journal.Journal[R] interface.
type JournalStub[R any] struct {
	journal.Journal[R]

	beforeWrite func(ctx context.Context, v uint64, r R) (bool, error)
	afterWrite  func(ctx context.Context, v uint64, r R, ok bool, err error) (bool, error)
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// If v > current then the behavior is undefined.
func (j *JournalStub[R]) Write(ctx context.Context, v uint64, r R) (ok bool, err error) {
	if j.beforeWrite != nil {
		ok, err := j.beforeWrite(ctx, v, r)
		if !ok || err != nil {
			return false, err
		}
	}

	defer func() {
		if j.afterWrite != nil {
			ok, err = j.afterWrite(ctx, v, r, ok, err)
		}
	}()

	if j.Journal != nil {
		return j.Journal.Write(ctx, v, r)
	}

	return false, nil
}

// FailOnceBeforeWrite configures s to return an error on the first call to
// s.Write() with a record that satisifies the given predicate function.
//
// The error is returned before the write is actually performed.
func FailOnceBeforeWrite[R any](
	s *JournalStub[R],
	pred func(R) bool,
) {
	var done uint32

	s.beforeWrite = func(ctx context.Context, v uint64, r R) (bool, error) {
		if pred(r) {
			if atomic.CompareAndSwapUint32(&done, 0, 1) {
				return false, errors.New("<error>")
			}
		}

		return true, nil
	}
}

// FailOnceAfterWrite configures s to return an error on the first call to
// s.Write() with a record that satisifies the given predicate function.
//
// The error is returned after the write is actually performed.
func FailOnceAfterWrite[R any](
	s *JournalStub[R],
	pred func(R) bool,
) {
	var done uint32

	s.afterWrite = func(
		ctx context.Context,
		v uint64,
		r R,
		ok bool,
		err error,
	) (bool, error) {
		if !ok || err != nil {
			return false, err
		}

		if pred(r) {
			if atomic.CompareAndSwapUint32(&done, 0, 1) {
				return false, errors.New("<error>")
			}
		}

		return true, nil
	}
}
