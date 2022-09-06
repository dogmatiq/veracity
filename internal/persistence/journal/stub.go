package journal

import (
	"context"
	"errors"
)

// OpenerStub is a test implementation of the Opener[R] interface.
type OpenerStub[R any] struct {
	Opener[R]

	OpenJournalFunc func(ctx context.Context, key string) (Journal[R], error)
}

// OpenJournal opens the journal identified by the given key.
func (o *OpenerStub[R]) OpenJournal(
	ctx context.Context,
	key string,
) (Journal[R], error) {
	if o.OpenJournalFunc != nil {
		return o.OpenJournalFunc(ctx, key)
	}

	if o.Opener != nil {
		return o.Opener.OpenJournal(ctx, key)
	}

	return nil, errors.New("<not implemented>")
}

// Stub is a test implementation of the Journal[R] interface.
type Stub[R any] struct {
	Journal[R]

	ReadFunc  func(ctx context.Context, v uint32) (R, bool, error)
	WriteFunc func(ctx context.Context, v uint32, r R) (bool, error)
	CloseFunc func() error
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *Stub[R]) Read(ctx context.Context, v uint32) (R, bool, error) {
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
func (j *Stub[R]) Write(ctx context.Context, v uint32, r R) (bool, error) {
	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, v, r)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, v, r)
	}

	return false, nil
}

// Close closes the journal.
func (j *Stub[R]) Close() error {
	if j.CloseFunc != nil {
		return j.CloseFunc()
	}

	if j.Journal != nil {
		return j.Journal.Close()
	}

	return nil
}
