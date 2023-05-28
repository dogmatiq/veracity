package memory

import (
	"context"
	"errors"
	"sync"
)

// FailBeforeKeyspaceSet configures the keyspace with the given name to return
// an error on the next call to Set() with a key/value pair that satisifies the
// given predicate function.
//
// The error is returned before the set is actually performed.
func FailBeforeKeyspaceSet(
	s *KeyValueStore,
	pred func(k, v []byte) bool,
	name string,
) {
	j, err := s.Open(context.Background(), name)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*keyspaceHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.BeforeSet = failSetOnce(pred)
}

// FailAfterKeyspaceSet configures the keyspace with the given name to return an
// error on the next call to Set() with a key/value pair that satisifies the
// given predicate function.
//
// The error is returned after the set is actually performed.
func FailAfterKeyspaceSet(
	s *KeyValueStore,
	pred func(k, v []byte) bool,
	name string,
) {
	j, err := s.Open(context.Background(), name)
	if err != nil {
		panic(err)
	}
	defer j.Close()

	h := j.(*keyspaceHandle)

	h.state.Lock()
	defer h.state.Unlock()

	h.state.AfterSet = failSetOnce(pred)
}

func failSetOnce(pred func(k, v []byte) bool) func(k, v []byte) error {
	var once sync.Once

	return func(k, v []byte) (err error) {
		if pred(k, v) {
			once.Do(func() {
				err = errors.New("<error>")
			})
		}

		return err
	}
}
