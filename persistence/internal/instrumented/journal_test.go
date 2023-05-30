package instrumented_test

import (
	"testing"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/internal/tlog"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	. "github.com/dogmatiq/veracity/persistence/internal/instrumented"
	"github.com/dogmatiq/veracity/persistence/journal"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestJournalStore(t *testing.T) {
	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{
				Next: &memory.JournalStore{},
				Telemetry: &telemetry.Provider{
					TracerProvider: trace.NewNoopTracerProvider(),
					MeterProvider:  noop.NewMeterProvider(),
					Logger:         tlog.New(t),
				},
			}
		},
	)
}
