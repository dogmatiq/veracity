package veracity

import (
	"testing"

	"github.com/dogmatiq/enginekit/enginetest"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"golang.org/x/exp/slog"
)

func TestEngine(t *testing.T) {
	logger := slog.Default()
	if testing.Verbose() {
		logger = test.NewLogger(t)
	}

	enginetest.RunTests(
		t,
		func(p enginetest.SetupParams) enginetest.SetupResult {
			e := New(
				p.App,
				WithJournalStore(&memory.JournalStore{}),
				WithKeyValueStore(&memory.KeyValueStore{}),
				WithLogger(logger),
			)

			return enginetest.SetupResult{
				RunEngine: e.Run,
				Executor:  e,
			}
		},
	)
}
