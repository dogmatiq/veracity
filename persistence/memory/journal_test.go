package memory_test

import (
	"testing"

	"github.com/dogmatiq/veracity/journal"
	. "github.com/dogmatiq/veracity/persistence/memory"
)

func TestJournal(t *testing.T) {
	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{}
		},
	)
}
