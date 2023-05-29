package instjournal_test

import (
	"testing"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/internal/tlog"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	. "github.com/dogmatiq/veracity/persistence/internal/instjournal"
	"github.com/dogmatiq/veracity/persistence/journal"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestStore(t *testing.T) {
	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &Store{
				Telemetry: &telemetry.Provider{
					TracerProvider: trace.NewNoopTracerProvider(),
					MeterProvider:  noop.MeterProvider{},
					Logger:         tlog.New(t),
				},
				Store: &memory.JournalStore{},
			}
		},
	)
}
