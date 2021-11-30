package persistence

import "context"

// A Journal is an append-only immutable sequence of records.
type Journal interface {
	// Open returns a reader used to read journal records in order, beginning at
	// the given record ID.
	//
	// If id is empty, the reader is opened at the first available record.
	Open(ctx context.Context, id string) (JournalReader, error)

	// Append adds a record to the end of the journal.
	//
	// lastID is the ID of the last record known to be in the journal. If it
	// does not match the ID of the last record, the append operation fails.
	Append(ctx context.Context, lastID string, data []byte) (id string, err error)
}

// A JournalReader is used to read journal record in order.
type JournalReader interface {
	// Next returns the next record in the journal or blocks until it becomes
	// available.
	//
	// If more is true there are guaranteed to be additional records in the
	// journal after the one returned. A value of false does NOT guarantee there
	// are no additional records.
	Next(ctx context.Context) (id string, data []byte, more bool, err error)

	// Close closes the reader.
	Close() error
}
