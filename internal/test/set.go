package test

import (
	"errors"

	"github.com/dogmatiq/persistencekit/driver/memory/memoryset"
)

// FailBeforeSetAdd configures the set with the given name to return an error on
// the next call to Add() with a value that satisifies the given predicate
// function.
//
// The error is returned before the add is actually performed.
func FailBeforeSetAdd(
	s *memoryset.BinaryStore,
	name string,
	pred func(v []byte) bool,
) {
	s.BeforeAdd = failAddOnce(name, pred)
}

// FailAfterSetAdd configures set with the given name to return an error on
// the next call to Add() with a value that satisifies the given predicate
// function.
//
// The error is returned after the add is actually performed.
func FailAfterSetAdd(
	s *memoryset.BinaryStore,
	name string,
	pred func(v []byte) bool,
) {
	s.AfterAdd = failAddOnce(name, pred)
}

func failAddOnce(
	name string,
	pred func(v []byte) bool,
) func(set string, v []byte) error {
	fail := FailOnce(errors.New("<error>"))

	return func(set string, v []byte) error {
		if set != name {
			return nil
		}

		if pred(v) {
			return fail()
		}

		return nil
	}
}
