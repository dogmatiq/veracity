package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"reflect"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Span represents a single named and timed operation of a workflow.
type Span struct {
	ctx               context.Context
	recorder          *Recorder
	name              string
	span              trace.Span
	attrs             []Attr
	logger            *slog.Logger
	groupedLogAttrs   []any
	ungroupedLogAttrs []any
	errors            int64
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
	}

	span.SetAttributes(r.attrs...)
	span.SetAttributes(attrs...)

	var spanLogAttrs []any

	if sctx := underlying.SpanContext(); sctx.HasSpanID() {
		spanLogAttrs = append(
			spanLogAttrs,
			slog.String("id", sctx.SpanID().String()),
		)
	}

	spanLogAttrs = append(
		spanLogAttrs,
		slog.String("name", name),
	)

	span.groupedLogAttrs = append(
		span.groupedLogAttrs,
		slog.Group("span", spanLogAttrs...),
	)

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
	s.span.SetAttributes(tel...)
	s.ungroupedLogAttrs = append(s.ungroupedLogAttrs, log...)
}

func (s *Span) resolveAttrs(attrs []Attr) ([]attribute.KeyValue, []any) {
	tel := make([]attribute.KeyValue, 0, len(attrs))
	log := make([]any, 0, len(attrs))

	prefix := attribute.Key(s.recorder.name + ".")

	for _, attr := range attrs {
		if attr, ok := attr.otel(); ok {
			attr.Key = prefix + attr.Key
			tel = append(tel, attr)
		}
		if attr, ok := attr.slog(); ok {
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

	eventAttrs, ungroupedLogAttrs := s.resolveAttrs(attrs)

	s.span.AddEvent(
		message,
		trace.WithAttributes(eventAttrs...),
	)

	ungroupedLogAttrs = append(ungroupedLogAttrs, s.ungroupedLogAttrs...)
	groupedLogAttrs := s.groupedLogAttrs

	if err != nil {
		exceptionAttrs := []any{
			slog.String("message", err.Error()),
		}

		if err := unwrapError(err); err != nil {
			exceptionAttrs = append(
				exceptionAttrs,
				slog.String("type", reflect.TypeOf(err).String()),
			)
		}

		groupedLogAttrs = append(
			groupedLogAttrs,
			slog.Group("exception", exceptionAttrs...),
		)
	}

	groupedLogAttrs = append(
		groupedLogAttrs,
		slog.Group(s.recorder.name, ungroupedLogAttrs...),
	)

	s.logger.Log(
		s.ctx,
		level,
		message,
		groupedLogAttrs...,
	)
}

var errorsStringType = reflect.TypeOf(errors.New(""))

// isStringError returns true if err is an error that was created using
// errors.New() or fmt.Errorf(), and therefore has no meaningful type.
func isStringError(err error) bool {
	return reflect.TypeOf(err) == errorsStringType
}

// unwrapError unwraps err until an error with a meaningful type is found. If
// all errors in the chain are "string errors", it returns nil.
func unwrapError(err error) error {
	for err != nil && isStringError(err) {
		err = errors.Unwrap(err)
	}
	return err
}
