package test

import (
	"reflect"

	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"google.golang.org/protobuf/proto"
)

// FailOnJournalOpen configures the journal with the given name to return an
// error on the next call to Open().
func FailOnJournalOpen[T any](
	s *memoryjournal.Store[T],
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
func FailBeforeJournalAppend[T any](
	s *memoryjournal.Store[T],
	name string,
	pred func(T) bool,
	err error,
) {
	s.BeforeAppend = failAppendOnce(name, pred, err)
}

// FailAfterJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned after the append is actually performed.
func FailAfterJournalAppend[T any](
	s *memoryjournal.Store[T],
	name string,
	pred func(T) bool,
	err error,
) {
	s.AfterAppend = failAppendOnce(name, pred, err)
}

func failAppendOnce[T any](
	name string,
	pred func(T) bool,
	err error,
) func(string, T) error {
	fail := FailOnce(err)

	return func(n string, rec T) error {
		if n != name {
			return nil
		}

		if pred(rec) {
			return fail()
		}

		return nil
	}
}

// XXX_FailBeforeJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned before the append is actually performed.
func XXX_FailBeforeJournalAppend[R proto.Message](
	s *memoryjournal.BinaryStore,
	name string,
	pred func(R) bool,
	err error,
) {
	s.BeforeAppend = XXX_failAppendOnce(name, pred, err)
}

// XXX_FailAfterJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned after the append is actually performed.
func XXX_FailAfterJournalAppend[R proto.Message](
	s *memoryjournal.BinaryStore,
	name string,
	pred func(R) bool,
	err error,
) {
	s.AfterAppend = XXX_failAppendOnce(name, pred, err)
}

func XXX_failAppendOnce[R proto.Message](
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
