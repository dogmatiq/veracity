package persistence

import "context"

// A Journal is an append-only immutable sequence of records.
type Journal interface {
	// Open returns a reader used to read journal records in order, beginning at
	// the given offset.
	Open(ctx context.Context, offset uint64) (Reader, error)

	// Append adds a record to the end of the journal.
	Append(ctx context.Context, data []byte) (offset uint64, _ error)
}

// A Reader is used to read journal record in order.
type Reader interface {
	// Next returns the next record in the journal or blocks until it becomes
	// available.
	Next(ctx context.Context) (offset uint64, data []byte, err error)

	// Close closes the reader.
	Close() error
}
