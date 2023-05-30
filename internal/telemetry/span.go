package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// Span represents a single named and timed operation of a workflow.
type Span struct {
	ctx      context.Context
	recorder *Recorder
	name     string
	span     trace.Span
	logger   *slog.Logger
	logAttrs []any
	errors   int64
}

// StartSpan starts a new span.
func (r *Recorder) StartSpan(
	ctx context.Context,
	name string,
	attrs ...Attr,
) (context.Context, *Span) {
	ctx, underlying := r.tracer.Start(
		ctx,
		name,
	)

	span := &Span{
		ctx:      ctx,
		recorder: r,
		name:     name,
		span:     underlying,
		logger:   r.logger,
		logAttrs: []any{
			slog.String("span.name", name),
		},
	}

	span.SetAttributes(r.attrs...)
	span.SetAttributes(attrs...)

	if sctx := underlying.SpanContext(); sctx.HasSpanID() {
		span.logAttrs = append(
			span.logAttrs,
			slog.String("span.id", sctx.SpanID().String()),
		)
	}

	return ctx, span
}

// End completes the span.
func (s *Span) End() {
	if s.errors == 0 {
		s.span.SetStatus(codes.Ok, "")
	} else {
		s.recorder.errors.Add(
			s.ctx,
			s.errors,
			metric.WithAttributes(
				attribute.String("span.name", s.name),
			),
		)
	}

	s.span.End()
}

// SetAttributes adds the given attributes to the underlying OpenTelemetry span
// and any future log messages.
func (s *Span) SetAttributes(attrs ...Attr) {
	tel, log := s.resolveAttrs(attrs)
	s.logAttrs = append(s.logAttrs, log...)
	s.span.SetAttributes(tel...)
}

func (s *Span) resolveAttrs(attrs []Attr) ([]attribute.KeyValue, []any) {
	tel := make([]attribute.KeyValue, 0, len(attrs))
	log := make([]any, 0, len(attrs))

	prefix := s.recorder.name + "."

	for _, attr := range attrs {
		if attr, ok := attr.otel(prefix); ok {
			tel = append(tel, attr)
		}
		if attr, ok := attr.slog(prefix); ok {
			log = append(log, attr)
		}
	}

	return tel, log
}

// Info logs an informational message to the log and as a span event.
func (s *Span) Info(message string, attrs ...Attr) {
	s.recordEvent(slog.LevelInfo, message, nil, attrs)
}

// Warn logs a warning message to the log and as a span event.
func (s *Span) Warn(message string, attrs ...Attr) {
	s.recordEvent(slog.LevelWarn, message, nil, attrs)
}

// Debug logs a debug message to the log and as a span event.
func (s *Span) Debug(message string, attrs ...Attr) {
	s.recordEvent(slog.LevelDebug, message, nil, attrs)
}

// Error logs an error message to the log and as a span event.
//
// It marks the span as an error and emits increments the "errors" metric.
func (s *Span) Error(message string, err error, attrs ...Attr) {
	s.recordEvent(slog.LevelError, message, err, attrs)
}

func (s *Span) recordEvent(
	level slog.Level,
	message string,
	err error,
	attrs []Attr,
) {
	if err != nil {
		s.span.SetStatus(codes.Error, err.Error())
		s.span.RecordError(err)
		s.errors++
	}

	if !s.logger.Enabled(s.ctx, level) {
		return
	}

	tel, log := s.resolveAttrs(attrs)

	s.span.AddEvent(
		message,
		trace.WithAttributes(tel...),
	)

	log = append(log, s.logAttrs...)

	s.logger.Log(
		s.ctx,
		level,
		message,
		log...,
	)
}
