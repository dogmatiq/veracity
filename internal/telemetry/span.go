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

	var extra []slog.Attr
	if sctx := span.SpanContext(); sctx.HasSpanID() {
		extra = append(
			extra,
			slog.String("span_id", sctx.SpanID().String()),
		)
	}

	return ctx, &Span{
		r,
		ctx,
		span,
		r.Logger.With(
			attrs.ForLogger(extra...)...,
		),
	}
}

// End completes the span.
func (s *Span) End() {
	s.span.End()
}

// SetAttributes sets attributes on the span.
//
// The attributes are set on the underlying OpenTelemetry span.
//
// Any future events recorded through s also inherit these attributes.
func (s *Span) SetAttributes(attributes ...Attr) {
	set := &attrSet{
		Namespace: s.recorder.Name,
		Attrs:     attributes,
	}

	s.span.SetAttributes(
		set.ForOpenTelemetry()...,
	)

	s.logger = s.logger.With(
		set.ForLogger()...,
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
