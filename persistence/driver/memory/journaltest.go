package memory

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

// FailBeforeJournalAppend configures the journal at the given path to return an
// error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned before the append is actually performed.
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

	h.state.BeforeAppend = failAppendOnce(pred)
}

// FailAfterJournalAppend configures the journal at the given path to return an
// error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned after the append is actually performed.
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

	h.state.AfterAppend = failAppendOnce(pred)
}

func failAppendOnce[R proto.Message](pred func(R) bool) func([]byte) error {
	var once sync.Once

	return func(data []byte) (err error) {
		var rec R
		rec = reflect.New(
			reflect.TypeOf(rec).Elem(),
		).Interface().(R)

		if err := proto.Unmarshal(data, rec); err != nil {
			panic(err)
		}

		if pred(rec) {
			once.Do(func() {
				err = errors.New("<error>")
			})
		}

		return err
	}
}
