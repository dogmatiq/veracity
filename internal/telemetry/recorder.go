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
// pkg is the path to the Go package that is performing the instrumentation. If
// it is an internal package, use the package path of the public parent package
// instead.
//
// name is the one-word name of the subsystem that the recorder is for, for
// example "journal" or "aggregate".
func (p *Provider) New(pkg, name string, attrs ...Attr) *Recorder {
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

// Recorder records traces, metrics and logs for a particular subsystem.
type Recorder struct {
	name   string
	attrs  []Attr
	tracer trace.Tracer
	meter  metric.Meter
	logger *slog.Logger
	errors metric.Int64Counter
}

// Int64Counter returns a new Int64Counter instrument.
func (r *Recorder) Int64Counter(name string, options ...metric.Int64CounterOption) metric.Int64Counter {
	return instrument(r, r.meter.Int64Counter, name, options)
}

// Int64UpDownCounter returns a new Int64UpDownCounter instrument.
func (r *Recorder) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) metric.Int64UpDownCounter {
	return instrument(r, r.meter.Int64UpDownCounter, name, options)
}

// Int64Histogram returns a new Int64Histogram instrument.
func (r *Recorder) Int64Histogram(name string, options ...metric.Int64HistogramOption) metric.Int64Histogram {
	return instrument(r, r.meter.Int64Histogram, name, options)
}

// Float64Counter returns a new Float64Counter instrument.
func (r *Recorder) Float64Counter(name string, options ...metric.Float64CounterOption) metric.Float64Counter {
	return instrument(r, r.meter.Float64Counter, name, options)
}

// Float64UpDownCounter returns a new Float64UpDownCounter instrument.
func (r *Recorder) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) metric.Float64UpDownCounter {
	return instrument(r, r.meter.Float64UpDownCounter, name, options)
}

// Float64Histogram returns a new Float64Histogram instrument.
func (r *Recorder) Float64Histogram(name string, options ...metric.Float64HistogramOption) metric.Float64Histogram {
	return instrument(r, r.meter.Float64Histogram, name, options)
}

// instrument constructs a new instrument using fn. Its name is prefixed with
// r's name.
func instrument[T, O any](
	r *Recorder,
	fn func(string, ...O) (T, error),
	name string,
	options []O,
) T {
	name = r.name + "." + name
	inst, err := fn(name, options...)
	if err != nil {
		panic(err)
	}
	return inst
}
