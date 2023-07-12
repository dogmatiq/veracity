package test

import "testing"

// TestingT is the subset of the [testing.TB] interface that is used by this
// package.
type TestingT interface {
	Log(...any)
	Logf(string, ...any)
	Fatal(...any)
	Fatalf(string, ...any)
	Error(...any)
	Errorf(string, ...any)

	Helper()
	Cleanup(func())
}

// FatalT is the subset of the [testing.TB] interface that is used by parts of
// this package that only need to cause tests to fail fatally.
type FatalT interface {
	Helper()
	Fatal(...any)
}

var _ TestingT = (testing.TB)(nil)
