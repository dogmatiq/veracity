package engineconfig

import (
	"github.com/dogmatiq/veracity/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	nooplog "go.opentelemetry.io/otel/log/noop"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func (c *Config) finalizeTelemetry() {
	if c.Telemetry.TracerProvider == nil {
		if c.UseEnv {
			c.Telemetry.TracerProvider = otel.GetTracerProvider()
		} else {
			c.Telemetry.TracerProvider = nooptrace.NewTracerProvider()
		}
	}

	if c.Telemetry.MeterProvider == nil {
		if c.UseEnv {
			c.Telemetry.MeterProvider = otel.GetMeterProvider()
		} else {
			c.Telemetry.MeterProvider = noopmetric.NewMeterProvider()
		}
	}

	if c.Telemetry.LoggerProvider == nil {
		if c.UseEnv {
			c.Telemetry.LoggerProvider = global.GetLoggerProvider()
		} else {
			c.Telemetry.LoggerProvider = nooplog.NewLoggerProvider()
		}
	}

	c.Telemetry.Attrs = append(
		c.Telemetry.Attrs,
		telemetry.UUID("node.id", c.NodeID),
	)
}
