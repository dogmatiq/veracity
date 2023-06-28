package prototest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func AssertEqual[T proto.Message](
	t *testing.T,
	actual, expect T,
	normalize ...func(T),
) {
	t.Helper()

	for _, fn := range normalize {
		fn(actual)
		fn(expect)
	}

	if diff := cmp.Diff(
		actual,
		expect,
		protocmp.Transform(),
	); diff != "" {
		t.Fatal(diff)
	}
}
