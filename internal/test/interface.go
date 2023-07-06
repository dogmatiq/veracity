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

var _ TestingT = (testing.TB)(nil)
