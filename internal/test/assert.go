package test

import (
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// Expect compares two values and fails the test if they are different.
func Expect[T any](
	t TestingT,
	got, want T,
	transforms ...func(T),
) {
	t.Helper()

	for _, fn := range transforms {
		fn(got)
		fn(want)
	}

	if diff := cmp.Diff(
		got,
		want,
		protocmp.Transform(),
	); diff != "" {
		t.Fatal(diff)
	}
}

// ExpectChannelToReceive waits until a value is received from a channel and
// then compares it to the expected value.
func ExpectChannelToReceive[T any](
	t TestingT,
	ch <-chan T,
	want T,
	transforms ...func(T),
) {
	t.Helper()

	ctx := contextOf(t)

	select {
	case <-ctx.Done():
		t.Fatalf("no value received on channel: %s", ctx.Err())
	case got, ok := <-ch:
		if ok {
			Expect(t, got, want, transforms...)
		} else {
			t.Error("channel closed while expecting to receive a value")
		}
	}
}

// ExpectChannelToClose waits until a channel is closed.
func ExpectChannelToClose[T any](
	t TestingT,
	ch <-chan T,
	transforms ...func(T),
) {
	t.Helper()

	ctx := contextOf(t)

	select {
	case <-ctx.Done():
		t.Fatalf("cannot was not closed: %s", ctx.Err())
	case got, ok := <-ch:
		if ok {
			t.Error("received a value while expecting channel to be closed")
			var want T // zero value
			Expect(t, got, want, transforms...)
		}
	}
}

// ExpectChannelToBlock expects reading from the channel to block until the
// given duration elapses.
func ExpectChannelToBlock[T any](
	t TestingT,
	d time.Duration,
	ch <-chan T,
	transforms ...func(T),
) {
	t.Helper()

	select {
	case <-time.After(d):
		// success! duration elapsed without receiving a value
	case got, ok := <-ch:
		if ok {
			t.Error("received a value while expecting channel to block")
			var want T // zero value
			Expect(t, got, want, transforms...)
		} else {
			t.Error("channel closed while expecting channel to block")
		}
	}
}
