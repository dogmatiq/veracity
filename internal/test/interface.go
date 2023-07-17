package test

import (
	"testing"

	"pgregory.net/rapid"
)

// TestingT is the subset of the [testing.TB] interface that is used by this
// package.
type TestingT interface {
	FailerT

	Helper()
	Cleanup(func())
}

// FailerT is the subset of the [testing.TB] interface that is used by parts of
// this package that only need to cause tests to fail.
type FailerT interface {
	Helper()
	Log(...any)
	Logf(string, ...any)
	Fatal(...any)
	Fatalf(string, ...any)
	Error(...any)
	Errorf(string, ...any)
}

var (
	_ TestingT = (testing.TB)(nil)
	_ FailerT  = (testing.TB)(nil)
	_ FailerT  = (*rapid.T)(nil)
)
