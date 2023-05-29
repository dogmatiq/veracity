package telemetry

import (
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/exp/slog"
)

// Info logs an info-level event.
func (s *Span) Info(message string, attributes ...Attr) {
	if !s.logger.Enabled(s.ctx, slog.LevelInfo) {
		return
	}

	attrs := s.eventAttrs(attributes)

	s.span.AddEvent(message, attrs.ForSpan())

	s.logger.InfoCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Warn logs an warning-level event.
func (s *Span) Warn(message string, attributes ...Attr) {
	if !s.logger.Enabled(s.ctx, slog.LevelWarn) {
		return
	}

	attrs := s.eventAttrs(attributes)

	s.span.AddEvent(message, attrs.ForSpan())

	s.logger.WarnCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Debug logs a debug-level event.
func (s *Span) Debug(message string, attributes ...Attr) {
	if !s.logger.Enabled(s.ctx, slog.LevelDebug) {
		return
	}

	attrs := s.eventAttrs(attributes)

	s.span.AddEvent(message, attrs.ForSpan())

	s.logger.DebugCtx(
		s.ctx,
		message,
		attrs.ForLogger()...,
	)
}

// Error logs an error-level event.
func (s *Span) Error(message string, err error, attributes ...Attr) {
	if !s.logger.Enabled(s.ctx, slog.LevelError) {
		return
	}

	attrs := s.eventAttrs(attributes)

	s.span.SetStatus(codes.Error, err.Error())
	s.span.RecordError(err, attrs.ForSpan())

	s.recorder.errors.Add(s.ctx, 1)

	s.logger.ErrorCtx(
		s.ctx,
		message,
		attrs.ForLogger(
			slog.String("error", err.Error()),
		)...,
	)
}

func (s *Span) eventAttrs(attributes []Attr) attrSet {
	return attrSet{
		Namespace: s.recorder.Name,
		Attrs:     attributes,
	}
}
