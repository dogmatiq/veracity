package journal_test

import (
	"testing"

	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/journaltest"
)

func TestInMemory(t *testing.T) {
	journaltest.RunTests(t, func() (journaltest.TestContext, error) {
		j := &journal.InMemory[[]byte]{}

		return journaltest.TestContext{
			Journal: j,
			Cleanup: j.Close,
		}, nil
	})
}
