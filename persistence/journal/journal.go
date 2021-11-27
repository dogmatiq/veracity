package journal

import "context"

// A Journal is an append-only immutable sequence of records.
type Journal interface {
	// Open returns a reader used to read journal records in order, beginning at
	// the given offset.
	Open(ctx context.Context, offset uint64) (Reader, error)

	// Append adds a record to the end of the journal.
	Append(ctx context.Context, rec []byte) (offset uint64, _ error)
}

// A Reader is used to read journal record in order.
type Reader interface {
	// Next returns the next record in the journal.
	//
	// It blocks until a record becomes available, an error occurs, or ctx is
	// canceled.
	Next(ctx context.Context) (offset uint64, rec []byte, _ error)

	// Close closes the reader.
	Close() error
}
