package journal

import (
	"context"
)

// A RangeFunc is a function used to range over the records in a [Journal].
//
// If err is non-nil, ranging stops and err is propagated up the stack.
// Otherwise, if ok is false, ranging stops without any error being propagated.
type RangeFunc func(ctx context.Context, offset uint64, rec []byte) (ok bool, err error)

// A Journal is an append-only log of binary records.
type Journal interface {
	// Bounds returns the half-open range [begin, end) describing the offsets of
	// the journal records that are available for reading.
	Bounds(ctx context.Context) (begin, end uint64, err error)

	// Get returns the record at the given offset.
	//
	// ok is false if the record does not exist, either because it has been
	// truncated or because the given offset has not been written yet.
	Get(ctx context.Context, offset uint64) (rec []byte, ok bool, err error)

	// Range invokes fn for each record in the journal, in order, starting with
	// the record at the given offset.
	Range(ctx context.Context, begin uint64, fn RangeFunc) error

	// RangeAll invokes fn for each record in the journal, in order.
	RangeAll(ctx context.Context, fn RangeFunc) error

	// Append adds a record to the journal.
	//
	// offset is the next "unused" offset in the journal. The first offset is
	// always 0.
	//
	// If there is already a record at the given offset then ok is false,
	// indicating an optimistic concurrency conflict.
	//
	// The behavior is undefined of the offset is larger than the next "unused"
	// offset.
	Append(ctx context.Context, offset uint64, rec []byte) (ok bool, err error)

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
	Truncate(ctx context.Context, end uint64) error

	// Close closes the journal.
	Close() error
}
