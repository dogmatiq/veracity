package journal

import (
	"context"
)

// A Journal is an append-only log that stores records of type R.
//
// The read/write operations are immediately consistent, meaning that successful
// writes are immediately visible to readers.
//
// Journals may be safely used concurrently.
type Journal[R any] interface {
	// Read returns the record that was written to produce the version v of the
	// journal.
	//
	// If the version does not exist ok is false.
	Read(ctx context.Context, v uint64) (r R, ok bool, err error)

	// Write appends a new record to the journal.
	//
	// v must be the current version of the journal.
	//
	// If v < current then the record is not persisted; ok is false indicating
	// an optimistic concurrency conflict.
	//
	// If v > current then the behavior is undefined.
	Write(ctx context.Context, v uint64, r R) (ok bool, err error)

	// Close closes the journal.
	Close() error
}

// Opener is an interface for opening journals by key.
type Opener[R any] interface {
	// OpenJournal opens the journal identified by the given key.
	OpenJournal(
		ctx context.Context,
		key string,
	) (Journal[R], error)
}
