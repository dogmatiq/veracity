package journal

import (
	"context"
)

// A Journal is an append-only log that stores records of type R.
//
// A journal's read/write operations are immediately consistent, meaning that
// the effect of a call to Write() or Truncate() is immediately observable by
// the Read() and ReadOldest() methods.
//
// Journals are safe for concurrent use.
type Journal[R any] interface {
	// Read returns the record written to produce the given version of the
	// journal.
	//
	// ok is false if the record does not exist, either because it has been
	// truncated or because the given version has not been written yet.
	Read(ctx context.Context, ver uint64) (rec R, ok bool, err error)

	// ReadOldest returns oldest record in the journal.
	//
	// ver is the version of the journal at which the record was written.
	//
	// ok is false if the journal is empty.
	ReadOldest(ctx context.Context) (ver uint64, rec R, ok bool, err error)

	// Write adds a record to the journal.
	//
	// ver is the next version of the journal. That is, the version to produce
	// as a result of writing this record. The first version is always 0.
	//
	// If the journal's current version >= ver then ok is false indicating an
	// optimistic concurrency conflict.
	//
	// If ver is greater than the "next" version the behavior is undefined.
	Write(ctx context.Context, ver uint64, rec R) (ok bool, err error)

	// Truncate removes the oldest records from the journal up to (but not
	// including) the record at the given offset.
	//
	// Passing an offset larger than the offset of the most recent record
	// results in undefined behavior.
	// Truncate(ctx context.Context, offset uint64) error

	// Close closes the journal.
	Close() error
}

// BinaryJournal is a journal that stores opaque binary records.
type BinaryJournal = Journal[[]byte]
