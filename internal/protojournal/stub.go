package protojournal

import (
	"errors"
	"reflect"

	"github.com/dogmatiq/veracity/journal/journaltest"
	"google.golang.org/protobuf/proto"
)

// FailBeforeWrite configures k to return an error on the first call to
// s.Write() with a record that satisifies the given predicate function.
//
// The error is returned before the write is actually performed.
func FailBeforeWrite[R proto.Message](
	j *journaltest.JournalStub,
	pred func(R) bool,
) {
	j.BeforeWrite = func(data []byte) error {
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
	j *journaltest.JournalStub,
	pred func(R) bool,
) {
	j.AfterWrite = func(data []byte) error {
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
