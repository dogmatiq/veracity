package journaltest

import (
	"context"
	"errors"
	"reflect"

	"github.com/dogmatiq/veracity/journal"
	"google.golang.org/protobuf/proto"
)

// JournalStub is a test implementation of the journal.Journal interface.
type JournalStub struct {
	journal.Journal

	before func([]byte) error
	after  func([]byte) error
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
	if j.before != nil {
		if err := j.before(rec); err != nil {
			return false, err
		}
	}

	if j.Journal != nil {
		ok, err := j.Journal.Write(ctx, ver, rec)
		if !ok || err != nil {
			return false, err
		}
	}

	if j.after != nil {
		if err := j.after(rec); err != nil {
			return false, err
		}
	}

	return true, nil
}

// FailBeforeWrite configures k to return an error on the first call to
// s.Write() with a record that satisifies the given predicate function.
//
// The error is returned before the write is actually performed.
func FailBeforeWrite[R proto.Message](
	j *JournalStub,
	pred func(R) bool,
) {
	j.before = func(data []byte) error {
		var rec R
		rec = reflect.New(
			reflect.TypeOf(rec).Elem(),
		).Interface().(R)

		if err := proto.Unmarshal(data, rec); err != nil {
			panic(err)
		}

		if pred(rec) {
			return errors.New("<error>")
		}

		return nil
	}
}

// FailAfterWrite configures j to return an error on the first call to
// j.Write() with a record that satisifies the given predicate function.
//
// The error is returned after the write is actually performed.
func FailAfterWrite[R proto.Message](
	j *JournalStub,
	pred func(R) bool,
) {
	j.after = func(data []byte) error {
		var rec R
		rec = reflect.New(
			reflect.TypeOf(rec).Elem(),
		).Interface().(R)

		if err := proto.Unmarshal(data, rec); err != nil {
			panic(err)
		}

		if pred(rec) {
			return errors.New("<error>")
		}

		return nil
	}
}
