package journaltest

import (
	"context"

	"github.com/dogmatiq/veracity/journal"
)

// JournalStub is a test implementation of the journal.Journal interface.
type JournalStub struct {
	journal.Journal

	BeforeWrite func([]byte) error
	AfterWrite  func([]byte) error
}

// Write adds a record to the journal.
//
// ver is the next version of the journal. That is, the version to produce as a
// result of writing this record. The first version is always 0.
//
// If the journal's current version >= ver then ok is false indicating an
// optimistic concurrency conflict.
//
// If ver is greater than the "next" version the behavior is undefined.
func (j *JournalStub) Write(ctx context.Context, ver uint64, rec []byte) (ok bool, err error) {
	if j.BeforeWrite != nil {
		if err := j.BeforeWrite(rec); err != nil {
			return false, err
		}
	}

	if j.Journal != nil {
		ok, err := j.Journal.Write(ctx, ver, rec)
		if !ok || err != nil {
			return false, err
		}
	}

	if j.AfterWrite != nil {
		if err := j.AfterWrite(rec); err != nil {
			return false, err
		}
	}

	return true, nil
}
