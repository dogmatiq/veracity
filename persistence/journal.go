package persistence

import "context"

// A Journal is an append-only sequence of opaque binary records.
type Journal interface {

	// Append adds a record to the end of the journal.
	//
	// prevID must be the ID of the most recent record, or an empty slice if the
	// journal is currently empty. If prevID matches the actual most recent
	// record, the record is appended and ok is true; otherwise, the append
	// operation is aborted and ok is false.
	//
	// id is the ID of the newly appended record.
	Append(ctx context.Context, prevID, rec []byte) (id []byte, ok bool, err error)

	// Truncate removes records from the beginning of the journal.
	//
	// keepID is the ID of the oldest record to keep. It becomes the record at
	// the start of the journal.
	Truncate(ctx context.Context, keepID []byte) error

	// NewReader returns a Reader that reads the records in the journal in the
	// order they were appended.
	//
	// If afterID is empty reading starts at the first record; otherwise,
	// reading starts at the record immediately after afterID.
	NewReader(ctx context.Context, afterID []byte, options JournalReaderOptions) (JournalReader, error)
}

// JournalReaderOptions controls the behavior of a JournalReader.
type JournalReaderOptions struct {
	// SkipTruncated indicates whether the reader should skip any truncated
	// records. By default the reader will return an error when attempting to
	// read a record that has been truncated.
	SkipTruncated bool
}

// A JournalReader reads records from a Journal in the order they were appended.
//
// It is not safe for concurrent use.
type JournalReader interface {
	// Next returns the next record in the journal.
	//
	// If ok is true, id is the ID of the next record and data is the record
	// data itself.
	//
	// If ok is false the end of the journal has been reached. The reader should
	// be closed and discarded; the behavior of subsequent calls to Next() is
	// undefined.
	Next(ctx context.Context) (id, data []byte, ok bool, err error)

	// Close closes the reader.
	Close() error
}
