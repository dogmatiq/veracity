package journaltest

import (
	"context"
	"errors"
	"reflect"

	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// JournalStub is a test implementation of the journal.Journal interface.
type JournalStub struct {
	journal.Journal

	before func([]byte) error
	after  func([]byte) error
}

// Append adds a record to the journal.
//
// ver is the next version of the journal. That is, the version to produce as a
// result of writing this record. The first version is always 0.
//
// If the journal's current version >= ver then ok is false indicating an
// optimistic concurrency conflict.
//
// If ver is greater than the "next" version the behavior is undefined.
func (j *JournalStub) Append(ctx context.Context, ver uint64, rec []byte) (ok bool, err error) {
	if j.before != nil {
		if err := j.before(rec); err != nil {
			return false, err
		}
	}

	if j.Journal != nil {
		ok, err := j.Journal.Append(ctx, ver, rec)
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

// FailBeforeAppend configures k to return an error on the first call to
// s.Append() with a record that satisifies the given predicate function.
//
// The error is returned before the append is actually performed.
func FailBeforeAppend[R proto.Message](
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

// FailAfterAppend configures j to return an error on the first call to
// j.Append() with a record that satisifies the given predicate function.
//
// The error is returned after the append is actually performed.
func FailAfterAppend[R proto.Message](
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
