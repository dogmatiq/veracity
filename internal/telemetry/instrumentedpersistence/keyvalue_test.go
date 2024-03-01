package instrumentedpersistence_test

import (
	"testing"

	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/veracity/internal/telemetry"
	. "github.com/dogmatiq/veracity/internal/telemetry/instrumentedpersistence"
	"github.com/dogmatiq/veracity/internal/test"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func TestKeyValueStore(t *testing.T) {
	kv.RunTests(
		t,
		func(t *testing.T) kv.Store {
			return &KeyValueStore{
				Next: &memorykv.Store{},
				Telemetry: &telemetry.Provider{
					TracerProvider: nooptrace.NewTracerProvider(),
					MeterProvider:  noopmetric.NewMeterProvider(),
					Logger:         test.NewLogger(t),
				},
			}
		},
	)
}
