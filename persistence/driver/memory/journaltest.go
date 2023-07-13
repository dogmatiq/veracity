package memory

import (
	"context"
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

// FailOnJournalOpen configures the journal with the given name to return an
// error on the next call to Open().
func FailOnJournalOpen(
	s *JournalStore,
	name string,
	produceErr error,
) {
	var once sync.Once

	s.onOpen = func(n string) error {
		var err error

		if name == n {
			once.Do(func() {
				err = produceErr
			})
		}

		return err
	}
}

// FailBeforeJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned before the append is actually performed.
func FailBeforeJournalAppend[R proto.Message](
	s *JournalStore,
	name string,
	pred func(R) bool,
	produceErr error,
) {
	j, err := s.Open(context.Background(), name)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*journalHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.BeforeAppend = failAppendOnce(pred, produceErr)
}

// FailAfterJournalAppend configures the journal with the given name to return
// an error on the next call to Append() with a record that satisifies the given
// predicate function.
//
// The error is returned after the append is actually performed.
func FailAfterJournalAppend[R proto.Message](
	s *JournalStore,
	name string,
	pred func(R) bool,
	produceErr error,
) {
	j, err := s.Open(context.Background(), name)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*journalHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.AfterAppend = failAppendOnce(pred, produceErr)
}

func failAppendOnce[R proto.Message](
	pred func(R) bool,
	produceErr error,
) func([]byte) error {
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
				err = produceErr
			})
		}

		return err
	}
}
