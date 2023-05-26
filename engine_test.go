package veracity

import (
	"log/slog"
	"testing"

	"github.com/dogmatiq/enginekit/enginetest"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
	"github.com/dogmatiq/spruce"
)

func TestEngine(t *testing.T) {
	logger := slog.Default()
	if testing.Verbose() {
		logger = spruce.NewLogger(t)
	}

	enginetest.RunTests(
		t,
		func(p enginetest.SetupParams) enginetest.SetupResult {
			e := New(
				p.App,
				WithJournalStore(&memoryjournal.BinaryStore{}),
				WithKeyValueStore(&memorykv.BinaryStore{}),
				WithLogger(logger),
			)

			return enginetest.SetupResult{
				RunEngine: e.Run,
				Executor:  e,
			}
		},
	)
}
