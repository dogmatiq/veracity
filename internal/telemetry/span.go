package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// Span represents a single named and timed operation of a workflow.
type Span struct {
	recorder *Recorder
	ctx      context.Context
	span     trace.Span
	logger   *slog.Logger
}

// StartSpan starts a new span.
func (r *Recorder) StartSpan(
	ctx context.Context,
	name string,
	options ...StartSpanOption,
) (context.Context, *Span) {
	var cfg spanConfig
	for _, opt := range options {
		opt.applyStartSpanOption(&cfg)
	}

	attrs := &attrSet{
		Namespace: r.Name,
		Attrs:     cfg.Attrs,
	}

	ctx, span := r.Tracer.Start(
		ctx,
		name,
		attrs.ForSpan(),
	)

	loggerAttrs := []any{
		attrs.ForLogger(),
		slog.String("span_name", name),
	}

	sctx := span.SpanContext()
	if sctx.HasSpanID() {
		loggerAttrs = append(
			loggerAttrs,
			slog.String("span_id", sctx.SpanID().String()),
		)
	}

	return ctx, &Span{
		r,
		ctx,
		span,
		r.Logger.With(loggerAttrs...),
	}
}

// End completes the span.
func (s *Span) End() {
	s.span.End()
}

// SetAttributes sets attributes on the span.
func (s *Span) SetAttributes(attrs ...Attr) {
	set := &attrSet{
		Namespace: s.recorder.Name,
		Attrs:     attrs,
	}

	s.span.SetAttributes(
		set.ForOpenTelemetryX()...,
	)
}

// StartSpanOption is an option that changes the behavior of a span.
type StartSpanOption interface {
	applyStartSpanOption(*spanConfig)
}

type spanConfig struct {
	Name  string
	Attrs []Attr
}

func (a Attr) applyStartSpanOption(c *spanConfig) {
	c.Attrs = append(c.Attrs, a)
}
