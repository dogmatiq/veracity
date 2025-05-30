package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// Span represents a single named and timed operation of a workflow.
type Span struct {
	span trace.Span
	end  func()
}

// StartSpan starts a new span and records the operation to the recorder's
// operation counter and "in-flight" gauge instruments.
func (r *Recorder) StartSpan(
	ctx context.Context,
	name string,
	attrs ...Attr,
) (context.Context, *Span) {
	ctx, underlying := r.tracer.Start(
		ctx,
		name,
		trace.WithAttributes(r.attrKVs.ToSlice()...),
		trace.WithAttributes(asAttrKeyValues(attrs)...),
	)

	op := String("operation", name)
	r.operationCount(ctx, 1, op)
	r.operationsInFlightCount(ctx, 1, op)

	return ctx, &Span{
		span: underlying,
		end:  func() { r.operationsInFlightCount(ctx, -1, op) },
	}
}

// End completes the span and decrements the "in-flight" gauge instrument.
func (s *Span) End() {
	s.end()
	s.span.End()
}

// SetAttributes adds the given attributes to the underlying OpenTelemetry span
// and any future log messages.
func (s *Span) SetAttributes(attrs ...Attr) {
	s.span.SetAttributes(asAttrKeyValues(attrs)...)
}
