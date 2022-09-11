package memory_test

import (
	"testing"

	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/journaltest"
	. "github.com/dogmatiq/veracity/persistence/memory"
)

func TestJournal(t *testing.T) {
	journaltest.RunTests(
		t,
		func(t *testing.T) journal.BinaryJournal {
			return &Journal[[]byte]{}
		},
	)
}
