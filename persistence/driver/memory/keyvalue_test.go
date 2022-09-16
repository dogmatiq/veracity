package memory_test

import (
	"testing"

	. "github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/dogmatiq/veracity/persistence/kv"
)

func TestKV(t *testing.T) {
	kv.RunTests(
		t,
		func(t *testing.T) kv.Store {
			return &KeyValueStore{}
		},
	)
}
