package telemetry

import (
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

// New returns a new Recorder instance.
//
// Name is the name of the subsystem that the recorder is for.
func (p *Provider) New(name string, options ...RecorderOption) *Recorder {
	cfg := recorderConfig{}
	for _, opt := range options {
		opt.applyRecorderOption(&cfg)
	}

	attrs := &attrSet{
		Namespace: name,
		Attrs:     cfg.Attrs,
	}

	tracer := p.TracerProvider.Tracer(
		name,
		tracerVersion,
		attrs.ForTracer(),
	)

	meter := p.MeterProvider.Meter(
		name,
		meterVersion,
		attrs.ForMeter(),
	)

	logger := p.Logger.With(
		attrs.ForLogger()...,
	)

	errorCount, err := meter.Int64Counter(
		"errors",
		metric.WithDescription("The number of errors that have occurred."),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		panic(err)
	}

	return &Recorder{
		Meter: meter,

		Name:   name,
		Tracer: tracer,
		Logger: logger,
		errors: errorCount,
	}
}
