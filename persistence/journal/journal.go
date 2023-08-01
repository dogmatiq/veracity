package journal

import (
	"context"
	"errors"
)

// Position is the index of a record within a [Journal]. The first record is always
// at position 0.
type Position uint64

// A RangeFunc is a function used to range over the records in a [Journal].
//
// If err is non-nil, ranging stops and err is propagated up the stack.
// Otherwise, if ok is false, ranging stops without any error being propagated.
type RangeFunc func(context.Context, Position, []byte) (ok bool, err error)

var (
	// ErrNotFound is returned by [Journal.Get] and [Journal.Range] if the
	// requested record does not exist, either because it has been truncated or
	// because the given position has not been written yet.
	ErrNotFound = errors.New("record not found")

	// ErrConflict is returned by [Journal.Append] if there is already a record at
	// the specified position.
	ErrConflict = errors.New("optimistic concurrency conflict")
)

// A Journal is an append-only log of binary records.
type Journal interface {
	// Bounds returns the half-open range [begin, end) describing the positions
	// of the first and last journal records that are available for reading.
	Bounds(ctx context.Context) (begin, end Position, err error)

	// Get returns the record at the given position.
	//
	// It returns [ErrNotFound] if there is no record at the given position.
	Get(ctx context.Context, pos Position) (rec []byte, err error)

	// Range invokes fn for each record in the journal, in order, starting with
	// the record at the given position.
	//
	// It returns [ErrNotFound] if there is no record at the given position.
	Range(ctx context.Context, pos Position, fn RangeFunc) error

	// Append adds a record to the journal.
	//
	// end must be the next "unused" position in the journal; the first position
	// is always 0.
	//
	// If there is already a record at the given position then [ErrConflict] is
	// returned, indicating an optimistic concurrency conflict.
	//
	// The behavior is undefined if end is greater than the next position.
	Append(ctx context.Context, end Position, rec []byte) error

	// Truncate removes journal records in the half-open range [0, end). That
	// is, it removes the oldest records up to, but not including, the record at
	// the given position.
	//
	// If it returns a non-nil error the truncation may have been partially
	// applied. That is, some of the records may have been removed but not all.
	// The implementation must guarantee that the oldest records are removed
	// first, such that there is never a "gap" between positions.
	//
	// The behavior is undefined if end is greater than the next position.
	Truncate(ctx context.Context, end Position) error

	// Close closes the journal.
	Close() error
}
