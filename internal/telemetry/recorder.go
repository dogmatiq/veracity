package telemetry

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

// Recorder records traces, metrics and logs for a particular subsystem.
type Recorder struct {
	Name   string
	Tracer trace.Tracer
	Meter  metric.Meter
	Logger *slog.Logger

	errors metric.Int64Counter
}

// RecorderOption is an option that changes the behavior of a recorder.
type RecorderOption interface {
	applyRecorderOption(*recorderConfig)
}

type recorderConfig struct {
	Name  string
	Attrs []Attr
}

func (a Attr) applyRecorderOption(c *recorderConfig) {
	c.Attrs = append(c.Attrs, a)
}
