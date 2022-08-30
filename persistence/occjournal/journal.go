package occjournal

import (
	"context"
	"errors"
)

// ErrConflict indicates that a journal record cannot be written because there
// is already a record at the given offset.
var ErrConflict = errors.New("optimistic concurrency conflict")

// Journal is an append-only log with version-based optimistic concurrency
// control.
type Journal[R any] interface {
	Read(ctx context.Context, version uint64) ([]R, uint64, error)
	Write(ctx context.Context, version uint64, entry R) error
}
