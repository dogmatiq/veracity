package test

import (
	"reflect"

	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"google.golang.org/protobuf/proto"
)

// FailOnJournalOpen configures the journal with the given name to return an
// error on the next call to Open().
func FailOnJournalOpen(
	s *memoryjournal.Store,
	name string,
	err error,
) {
	fail := FailOnce(err)
	s.BeforeOpen = func(n string) error {
		if name == n {
			return fail()
		}
		return nil
	}
}

// FailBeforeJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned before the append is actually performed.
func FailBeforeJournalAppend[R proto.Message](
	s *memoryjournal.Store,
	name string,
	pred func(R) bool,
	err error,
) {
	s.BeforeAppend = failAppendOnce(name, pred, err)
}

// FailAfterJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned after the append is actually performed.
func FailAfterJournalAppend[R proto.Message](
	s *memoryjournal.Store,
	name string,
	pred func(R) bool,
	err error,
) {
	s.AfterAppend = failAppendOnce(name, pred, err)
}

func failAppendOnce[R proto.Message](
	name string,
	pred func(R) bool,
	err error,
) func(string, []byte) error {
	fail := FailOnce(err)

	return func(n string, data []byte) error {
		if n != name {
			return nil
		}

		var rec R
		rec = reflect.New(
			reflect.TypeOf(rec).Elem(),
		).Interface().(R)

		if err := proto.Unmarshal(data, rec); err != nil {
			panic(err)
		}

		if pred(rec) {
			return fail()
		}
		return nil
	}
}