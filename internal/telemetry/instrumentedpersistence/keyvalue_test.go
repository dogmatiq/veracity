package instrumentedpersistence_test

import (
	"testing"

	"github.com/dogmatiq/veracity/internal/telemetry"
	. "github.com/dogmatiq/veracity/internal/telemetry/instrumentedpersistence"
	"github.com/dogmatiq/veracity/internal/tlog"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/dogmatiq/veracity/persistence/kv"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func TestKeyValueStore(t *testing.T) {
	kv.RunTests(
		t,
		func(t *testing.T) kv.Store {
			return &KeyValueStore{
				Next: &memory.KeyValueStore{},
				Telemetry: &telemetry.Provider{
					TracerProvider: trace.NewNoopTracerProvider(),
					MeterProvider:  noop.NewMeterProvider(),
					Logger:         tlog.New(t),
				},
			}
		},
	)
}
