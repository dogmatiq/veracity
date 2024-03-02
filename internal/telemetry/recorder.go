package telemetry

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

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
