package testutil

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// Expect compares two values and fails the test if they are different.
func Expect[T any](
	t *testing.T,
	got, want T,
	normalizers ...func(T),
) {
	t.Helper()

	for _, fn := range normalizers {
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

// ExpectToReceive waits until a value is received from a channel and then
// compares it to the expected value.
func ExpectToReceive[T any](
	ctx context.Context,
	t *testing.T,
	got <-chan T,
	want T,
	normalizers ...func(T),
) {
	t.Helper()

	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case actual := <-got:
		Expect(t, actual, want, normalizers...)
	}
}
