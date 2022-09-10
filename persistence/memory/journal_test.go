package memory_test

import (
	"testing"

	"github.com/dogmatiq/veracity/journal/journaltest"
	. "github.com/dogmatiq/veracity/persistence/memory"
)

func TestJournal(t *testing.T) {
	journaltest.RunTests(t, func() (journaltest.TestContext, error) {
		j := &Journal[[]byte]{}

		return journaltest.TestContext{
			Journal: j,
			Cleanup: j.Close,
		}, nil
	})
}
