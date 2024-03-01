package instrumentedpersistence_test

import (
	"testing"

	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/telemetry"
	. "github.com/dogmatiq/veracity/internal/telemetry/instrumentedpersistence"
	"github.com/dogmatiq/veracity/internal/test"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func TestJournalStore(t *testing.T) {
	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{
				Next: &memoryjournal.Store{},
				Telemetry: &telemetry.Provider{
					TracerProvider: nooptrace.NewTracerProvider(),
					MeterProvider:  noopmetric.NewMeterProvider(),
					Logger:         test.NewLogger(t),
				},
			}
		},
	)
}
