package veracity

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// WithTracerProvider is an EngineOption that configures the engine to use the
// given OpenTelemetry tracer provider for recording spans.
func WithTracerProvider(p trace.TracerProvider) EngineOption {
	if p == nil {
		panic("tracer provider must not be nil")
	}

	return option{
		engineOption: func(e *Engine) {
			e.telemetry.TracerProvider = p
		},
	}
}

// WithMetricProvider is an EngineOption that configures the engine to use the
// given OpenTelemetry metric provider for recording metrics.
func WithMetricProvider(p metric.MeterProvider) EngineOption {
	if p == nil {
		panic("metric provider must not be nil")
	}

	return option{
		engineOption: func(e *Engine) {
			e.telemetry.MeterProvider = p
		},
	}
}

// WithLogger is an EngineOption that configures the engine to log to the given
// slog logger.
func WithLogger(l *slog.Logger) EngineOption {
	if l == nil {
		panic("logger must not be nil")
	}

	return option{
		engineOption: func(e *Engine) {
			e.telemetry.Logger = l
		},
	}
}
