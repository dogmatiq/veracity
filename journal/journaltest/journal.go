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

	beforeWrite func(R) error
	afterWrite  func(R) error
}

// Write adds a record to the journal.
//
// ver is the next version of the journal. That is, the version to produce as a
// result of writing this record. The first version is always 0.
//
// If the journal's current version >= ver then ok is false indicating an
// optimistic concurrency conflict.
//
// If ver is greater than the "next" version the behavior is undefined.
func (j *JournalStub[R]) Write(ctx context.Context, ver uint64, rec R) (ok bool, err error) {
	if j.beforeWrite != nil {
		if err := j.beforeWrite(rec); err != nil {
			return false, err
		}
	}

	if j.Journal != nil {
		ok, err := j.Journal.Write(ctx, ver, rec)
		if !ok || err != nil {
			return false, err
		}
	}

	if j.afterWrite != nil {
		if err := j.afterWrite(rec); err != nil {
			return false, err
		}
	}

	return true, nil
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

	s.beforeWrite = func(rec R) error {
		if pred(rec) {
			if atomic.CompareAndSwapUint32(&done, 0, 1) {
				return errors.New("<error>")
			}
		}

		return nil
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

	s.afterWrite = func(rec R) error {
		if pred(rec) {
			if atomic.CompareAndSwapUint32(&done, 0, 1) {
				return errors.New("<error>")
			}
		}

		return nil
	}
}
