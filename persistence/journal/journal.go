package journal

import (
	"context"
	"errors"
)

// Offset is the offset of a record within a [Journal]. The first record is
// always at offset 0.
type Offset uint64

// A RangeFunc is a function used to range over the records in a [Journal].
//
// If err is non-nil, ranging stops and err is propagated up the stack.
// Otherwise, if ok is false, ranging stops without any error being propagated.
type RangeFunc func(context.Context, Offset, []byte) (ok bool, err error)

// ErrConflict is returned by [Journal.Append] if there is already a record at
// the specified offset.
var ErrConflict = errors.New("optimistic concurrency conflict")

// A Journal is an append-only log of binary records.
type Journal interface {
	// Bounds returns the half-open range [begin, end) describing the offsets of
	// the journal records that are available for reading.
	Bounds(ctx context.Context) (begin, end Offset, err error)

	// Get returns the record at the given offset.
	//
	// ok is false if the record does not exist, either because it has been
	// truncated or because the given offset has not been written yet.
	Get(ctx context.Context, off Offset) (rec []byte, ok bool, err error)

	// Range invokes fn for each record in the journal, in order, starting with
	// the record at the given offset.
	Range(ctx context.Context, begin Offset, fn RangeFunc) error

	// RangeAll invokes fn for each record in the journal, in order.
	RangeAll(ctx context.Context, fn RangeFunc) error

	// Append adds a record to the journal.
	//
	// end must be the next "unused" offset in the journal. The first offset is
	// always 0.
	//
	// If there is already a record at the given offset then [ErrConflict] is
	// returned, indicating an optimistic concurrency conflict.
	//
	// If end is greater than the next offset the behavior is undefined.
	Append(ctx context.Context, end Offset, rec []byte) error

	// Truncate removes journal records in the half-open range [..., end). That
	// is, it removes the oldest records up to, but not including, the record at
	// the given offset.
	//
	// If it returns a non-nil error the truncation may have been partially
	// applied. That is, some of the records may have been removed but not all.
	// The implementation must guarantee that the oldest records are removed
	// first, such that there is never a "gap" between offsets.
	//
	// The behavior is undefined if the offset is larger than the next "unused"
	// offset.
	Truncate(ctx context.Context, end Offset) error

	// Close closes the journal.
	Close() error
}
