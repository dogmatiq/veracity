package journal

import (
	"context"
)

// Journal is an append-only log that stores records of type R.
//
// Journals may be safely used concurrently.
type Journal[R any] interface {
	// Read returns the record that was written to produce the given version of
	// the journal.
	//
	// If the version does not exist ok is false.
	Read(ctx context.Context, ver uint64) (rec R, ok bool, err error)

	// Write appends a new record to the journal.
	//
	// ver must be the current version of the journal.
	//
	// If ver < current then the record is not persisted; ok is false indicating
	// an optimistic concurrency conflict.
	//
	// If ver > current then the behavior is undefined.
	Write(ctx context.Context, ver uint64, rec R) (ok bool, err error)
}

// BinaryJournal is an append-only log that stores binary records.
type BinaryJournal interface {
	Journal[[]byte]
}
