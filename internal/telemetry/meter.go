package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// Instrument is a function that records a metric value of type T.
type Instrument[T any] func(context.Context, T, ...Attr)

// Counter returns a new monotonic counter instrument.
func (r *Recorder) Counter(name, unit, desc string) Instrument[int64] {
	inst, err := r.meter.Int64Counter(
		name,
		metric.WithUnit(unit),
		metric.WithDescription(desc),
	)
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, value int64, attrs ...Attr) {
		inst.Add(
			ctx,
			value,
			metric.WithAttributeSet(r.attrKVs),
			metric.WithAttributes(asAttrKeyValues(attrs)...),
		)
	}
}

// UpDownCounter returns a new counter instrument that can increase or decrease.
func (r *Recorder) UpDownCounter(name, unit, desc string) Instrument[int64] {
	inst, err := r.meter.Int64UpDownCounter(
		name,
		metric.WithUnit(unit),
		metric.WithDescription(desc),
	)
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, value int64, attrs ...Attr) {
		inst.Add(
			ctx,
			value,
			metric.WithAttributeSet(r.attrKVs),
			metric.WithAttributes(asAttrKeyValues(attrs)...),
		)
	}
}

// Histogram returns a new histogram instrument.
func (r *Recorder) Histogram(name, unit, desc string) Instrument[int64] {
	inst, err := r.meter.Int64Histogram(
		name,
		metric.WithUnit(unit),
		metric.WithDescription(desc),
	)
	if err != nil {
		panic(err)
	}

	return func(ctx context.Context, value int64, attrs ...Attr) {
		inst.Record(
			ctx,
			value,
			metric.WithAttributeSet(r.attrKVs),
			metric.WithAttributes(asAttrKeyValues(attrs)...),
		)
	}
}

var (
	// ReadDirection is an attribute that indicates a read operation.
	ReadDirection = String("network.io.direction", "read")

	// WriteDirection is an attribute that indicates a write operation.
	WriteDirection = String("network.io.direction", "write")
)
