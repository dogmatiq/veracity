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

	ReadFunc func(ctx context.Context, v uint32) (R, bool, error)

	WriteFunc   func(ctx context.Context, v uint32, r R) (bool, error)
	beforeWrite func(ctx context.Context, v uint32, r R) (bool, error)
	afterWrite  func(ctx context.Context, v uint32, r R, ok bool, err error) (bool, error)

	CloseFunc func() error
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *JournalStub[R]) Read(ctx context.Context, v uint32) (R, bool, error) {
	if j.ReadFunc != nil {
		return j.ReadFunc(ctx, v)
	}

	if j.Journal != nil {
		return j.Journal.Read(ctx, v)
	}

	var zero R
	return zero, false, nil
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// If v > current then the behavior is undefined.
func (j *JournalStub[R]) Write(ctx context.Context, v uint32, r R) (ok bool, err error) {
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

	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, v, r)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, v, r)
	}

	return false, nil
}

// Close closes the journal.
func (j *JournalStub[R]) Close() error {
	if j.CloseFunc != nil {
		return j.CloseFunc()
	}

	if j.Journal != nil {
		return j.Journal.Close()
	}

	return nil
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

	s.beforeWrite = func(ctx context.Context, v uint32, r R) (bool, error) {
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
		v uint32,
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
