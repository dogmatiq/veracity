package memory_test

import (
	"testing"

	. "github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/dogmatiq/veracity/persistence/journal"
)

func TestJournal(t *testing.T) {
	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{}
		},
	)
}
