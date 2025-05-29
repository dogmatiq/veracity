package test

import (
	"github.com/dogmatiq/spruce"
	"github.com/dogmatiq/veracity/internal/telemetry"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// NewTelemetryProvider returns a new telemetry provider for use in tests.
func NewTelemetryProvider(t TestingT) *telemetry.Provider {
	t.Helper()

	return &telemetry.Provider{
		TracerProvider: nooptrace.NewTracerProvider(),
		MeterProvider:  noopmetric.NewMeterProvider(),
		Logger:         spruce.NewTestLogger(t),
	}

}
