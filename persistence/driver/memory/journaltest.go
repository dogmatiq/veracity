package memory

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

// FailBeforeJournalAppend configures j to return an error on the first call to
// s.Append() with a record that satisifies the given predicate function.
//
// The error is returned before the append is actually performed.
//
// It panics if j is not an in-memory journal.
func FailBeforeJournalAppend[R proto.Message](
	s *JournalStore,
	pred func(R) bool,
	path ...string,
) {
	j, err := s.Open(context.Background(), path...)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*journalHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.BeforeAppend = failOnce(pred)
}

// FailAfterJournalAppend configures j to return an error on the first call to
// j.Append() with a record that satisifies the given predicate function.
//
// The error is returned after the append is actually performed.
//
// It panics if j is not an in-memory journal.
func FailAfterJournalAppend[R proto.Message](
	s *JournalStore,
	pred func(R) bool,
	path ...string,
) {
	j, err := s.Open(context.Background(), path...)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*journalHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.AfterAppend = failOnce(pred)
}

func failOnce[R proto.Message](pred func(R) bool) func([]byte) error {
	var once sync.Once

	return func(data []byte) error {
		var rec R
		rec = reflect.New(
			reflect.TypeOf(rec).Elem(),
		).Interface().(R)

		if err := proto.Unmarshal(data, rec); err != nil {
			panic(err)
		}

		var err error

		if pred(rec) {
			once.Do(func() {
				err = errors.New("<error>")
			})
		}

		return err
	}
}
