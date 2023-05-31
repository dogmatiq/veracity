package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// Provider provides Recorder instances scoped to particular subsystems.
type Provider struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// DefaultProvider returns a telemetry provider that uses the global
// OpenTelemetry providers and the default logger.
func DefaultProvider() *Provider {
	return &Provider{
		TracerProvider: otel.GetTracerProvider(),
		MeterProvider:  otel.GetMeterProvider(),
		Logger:         slog.Default(),
	}
}

// Recorder returns a new Recorder instance.
//
// pkg is the path to the Go package that is performing the instrumentation. If
// it is an internal package, use the package path of the public parent package
// instead.
//
// name is the one-word name of the subsystem that the recorder is for, for
// example "journal" or "aggregate".
func (p *Provider) Recorder(pkg, name string, attrs ...Attr) *Recorder {
	r := &Recorder{
		name:   "io.dogmatiq.veracity." + name,
		attrs:  attrs,
		tracer: p.TracerProvider.Tracer(pkg, tracerVersion),
		meter:  p.MeterProvider.Meter(pkg, meterVersion),
		logger: p.Logger,
	}

	r.errors = r.Int64Counter(
		"errors",
		metric.WithDescription("The number of errors that have occurred."),
		metric.WithUnit("{error}"),
	)

	return r
}
