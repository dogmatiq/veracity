package test

import (
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

// Expect compares two values and fails the test if they are different.
func Expect[T any](
	t FailerT,
	failMessage string,
	got, want T,
	transforms ...func(T) T,
) {
	t.Helper()

	for _, fn := range transforms {
		got = fn(got)
		want = fn(want)
	}

	if diff := cmp.Diff(
		want,
		got,
		protocmp.Transform(),
		cmpopts.EquateEmpty(),
		cmpopts.EquateErrors(),
	); diff != "" {
		t.Log(failMessage)
		t.Fatal(diff)
	}
}

// ExpectChannelToReceive waits until a value is received from a channel and
// then compares it to the expected value.
func ExpectChannelToReceive[T any](
	t FailerT,
	ch <-chan T,
	want T,
	transforms ...func(T) T,
) {
	t.Helper()

	ctx := contextOf(t)

	select {
	case <-ctx.Done():
		t.Fatalf("no value received on channel: %s", ctx.Err())
	case got, ok := <-ch:
		if ok {
			Expect(
				t,
				"channel received an unexpected value",
				got,
				want,
				transforms...,
			)
		} else {
			t.Error("channel closed while expecting to receive a value")
		}
	}
}

// ExpectChannelToClose waits until a channel is closed.
func ExpectChannelToClose[T any](
	t TestingT,
	ch <-chan T,
	transforms ...func(T) T,
) {
	t.Helper()

	ctx := contextOf(t)

	select {
	case <-ctx.Done():
		t.Fatalf("cannot was not closed: %s", ctx.Err())
	case got, ok := <-ch:
		if ok {
			var want T // zero value
			Expect(
				t,
				"channel received a value while expecting channel to be closed",
				got,
				want,
				transforms...,
			)
		}
	}
}

// ExpectChannelToBlockForDuration expects reading from the channel to block
// until the given duration elapses.
func ExpectChannelToBlockForDuration[T any](
	t TestingT,
	d time.Duration,
	ch <-chan T,
	transforms ...func(T) T,
) {
	t.Helper()

	select {
	case <-time.After(d):
		// success! duration elapsed without receiving a value
	case got, ok := <-ch:
		if ok {
			var want T // zero value
			Expect(t,
				"channel received a value while expecting channel to block",
				got,
				want,
				transforms...,
			)
		} else {
			t.Error("channel closed while expecting channel to block")
		}
	}
}

// ExpectChannelWouldBlock expects reading from the channel would block.
func ExpectChannelWouldBlock[T any](
	t TestingT,
	ch <-chan T,
	transforms ...func(T) T,
) {
	t.Helper()

	select {
	default:
		// success! there is no value available on the channel
	case got, ok := <-ch:
		if ok {
			var want T // zero value
			Expect(t,
				"channel received a value while expecting channel to block",
				got,
				want,
				transforms...,
			)
		} else {
			t.Error("channel closed while expecting channel to block")
		}
	}
}
