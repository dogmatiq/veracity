package journal

import (
	"context"
)

// Journal is a permanent record of application state changes.
type Journal interface {
	// Append adds a record to the end of the journal.
	//
	// prevID must be the ID of the most recent record, or an empty slice if the
	// journal is currently empty; otherwise, the append operation fails.
	//
	// It returns the ID of the newly appended record.
	Append(ctx context.Context, prevID []byte, rec Record) (id []byte, err error)

	// Open returns a Reader that reads the records in the journal in the order
	// they were appended.
	//
	// If afterID is empty reading starts at the first record; otherwise,
	// reading starts at the record immediately after afterID.
	Open(ctx context.Context, afterID []byte) (Reader, error)
}

// A Reader reads records from a journal in the order they were appended.
//
// It is not safe for concurrent use.
type Reader interface {
	// Next returns the next record in the journal.
	//
	// If ok is true, id is the ID of the next record and rec is the record
	// itself.
	//
	// If ok is false the end of the journal has been reached. The reader should
	// be closed and discarded; the behavior of subsequent calls to Next() is
	// undefined.
	Next(ctx context.Context) (id []byte, rec Record, ok bool, err error)

	// Close closes the reader.
	Close() error
}
