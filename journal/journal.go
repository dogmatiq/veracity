package journal

import (
	"context"
	"errors"
)

// ErrConflict indicates that a journal cannot be written because there is
// already an entry at the given offset.
var ErrConflict = errors.New("optimistic concurrency conflict")

// Journal is an append-only log.
type Journal[E any] interface {
	Read(ctx context.Context, offset *uint64) ([]E, error)
	Write(ctx context.Context, offset uint64, entry E) error
}
