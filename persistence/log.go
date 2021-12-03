package persistence

import "context"

// A Log is an append-only sequence of immutable, opaque binary records.
type Log interface {
	// Append adds a record to the end of the log.
	//
	// prevID must be the ID of the most recent record, or an empty slice if the
	// log is currently empty; otherwise, the append operation fails.
	//
	// It returns the ID of the newly appended record.
	Append(ctx context.Context, prevID, data []byte) (id []byte, err error)

	// Open returns a Reader that reads the records in the log in the order they
	// were appended.
	//
	// If afterID is empty reading starts at the first record; otherwise,
	// reading starts at the record immediately after afterID.
	Open(ctx context.Context, afterID []byte) (LogReader, error)
}

// A LogReader reads records from a Log in the order they were appended.
//
// It is not safe for concurrent use.
type LogReader interface {
	// Next returns the next record in the log.
	//
	// If ok is true, id is the ID of the next record and data is the record
	// data itself.
	//
	// If ok is false the end of the log has been reached. The reader should be
	// closed and discarded; the behavior of subsequent calls to Next() is
	// undefined.
	Next(ctx context.Context) (id, data []byte, ok bool, err error)

	// Close closes the reader.
	Close() error
}
