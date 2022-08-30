package occjournal

import (
	"context"
	"errors"
)

// ErrConflict indicates that a journal record cannot be written because there
// is already a record at the given offset.
var ErrConflict = errors.New("optimistic concurrency conflict")

// Journal is an append-only log that stores records of type R.
type Journal[R any] interface {
	Read(ctx context.Context, ver uint64) ([]R, uint64, error)
	Write(ctx context.Context, ver uint64, rec R) error
}
