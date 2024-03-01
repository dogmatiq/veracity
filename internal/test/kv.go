package test

import (
	"errors"

	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
)

// FailBeforeKeyspaceSet configures the keyspace with the given name to return
// an error on the next call to Set() with a key/value pair that satisifies the
// given predicate function.
//
// The error is returned before the set is actually performed.
func FailBeforeKeyspaceSet(
	s *memorykv.Store,
	name string,
	pred func(k, v []byte) bool,
) {
	s.BeforeSet = failSetOnce(name, pred)
}

// FailAfterKeyspaceSet configures the keyspace with the given name to return an
// error on the next call to Set() with a key/value pair that satisifies the
// given predicate function.
//
// The error is returned after the set is actually performed.
func FailAfterKeyspaceSet(
	s *memorykv.Store,
	name string,
	pred func(k, v []byte) bool,
) {
	s.AfterSet = failSetOnce(name, pred)
}

func failSetOnce(
	name string,
	pred func(k, v []byte) bool,
) func(ks string, k, v []byte) error {
	fail := FailOnce(errors.New("<error>"))

	return func(ks string, k, v []byte) error {
		if ks != name {
			return nil
		}

		if pred(k, v) {
			return fail()
		}

		return nil
	}
}
