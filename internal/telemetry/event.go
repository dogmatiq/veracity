package telemetry

import (
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/exp/slog"
)

// Info logs an info-level event.
func (s *Span) Info(message string, options ...EventOption) {
	if !s.logger.Enabled(s.ctx, slog.LevelInfo) {
		return
	}

	attrs := s.eventAttrs(options)

	s.span.AddEvent(message, attrs.ForSpan())

	s.logger.InfoCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Warn logs an warning-level event.
func (s *Span) Warn(message string, options ...EventOption) {
	if !s.logger.Enabled(s.ctx, slog.LevelWarn) {
		return
	}

	attrs := s.eventAttrs(options)

	s.span.AddEvent(message, attrs.ForSpan())

	s.logger.WarnCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Debug logs a debug-level event.
func (s *Span) Debug(message string, options ...EventOption) {
	if !s.logger.Enabled(s.ctx, slog.LevelDebug) {
		return
	}

	attrs := s.eventAttrs(options)

	s.span.AddEvent(
		message,
		attrs.ForSpan(),
	)

	s.logger.DebugCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Error logs an error-level event.
func (s *Span) Error(message string, err error, options ...EventOption) {
	if !s.logger.Enabled(s.ctx, slog.LevelError) {
		return
	}

	attrs := s.eventAttrs(options)

	s.span.SetStatus(codes.Error, err.Error())
	s.span.RecordError(err, attrs.ForSpan())

	s.recorder.errors.Add(s.ctx, 1)

	s.logger.ErrorCtx(
		s.ctx,
		message,
		append(
			[]any{slog.String("error", err.Error())},
			attrs.ForLogger()...,
		)...,
	)
}

// EventOption is an option that changes the behavior of an event.
type EventOption interface {
	applyEventOption(*eventConfig)
}

type eventConfig struct {
	Attrs []Attr
}

func (a Attr) applyEventOption(c *eventConfig) {
	c.Attrs = append(c.Attrs, a)
}

func (s *Span) eventAttrs(options []EventOption) attrSet {
	var cfg eventConfig
	for _, opt := range options {
		opt.applyEventOption(&cfg)
	}

	return attrSet{
		Namespace: s.recorder.Name,
		Attrs:     cfg.Attrs,
	}
}
