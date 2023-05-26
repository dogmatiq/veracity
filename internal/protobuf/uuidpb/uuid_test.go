package uuidpb_test

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/dogmatiq/veracity/internal/protobuf/uuidpb"
	"github.com/google/uuid"
)

func TestUUID_ToNative(t *testing.T) {
	t.Parallel()

	expect := uuid.New()
	pb := FromNative(expect)
	actual := pb.ToNative()

	if expect != actual {
		t.Fatalf("got %s, want %s", actual, expect)
	}
}

func TestUUID_ToString(t *testing.T) {
	t.Parallel()

	expect := uuid.New()
	pb := FromNative(expect)
	actual := pb.ToString()

	if expect.String() != actual {
		t.Fatalf("got %s, want %s", actual, expect)
	}
}

func TestUUID_ToBytes(t *testing.T) {
	t.Parallel()

	expect := uuid.New()
	pb := FromNative(expect)
	actual := pb.ToBytes()

	if !bytes.Equal(actual, expect[:]) {
		t.Fatalf("got %#v, want %#v", actual, expect)
	}
}

func TestUUID_Format(t *testing.T) {
	t.Parallel()

	pb := New()
	expect := pb.ToNative().String()
	actual := fmt.Sprintf("%s", pb)

	if expect != actual {
		t.Fatalf("got %#v, want %#v", actual, expect)
	}
}
