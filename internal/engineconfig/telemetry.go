package engineconfig

import (
	"log/slog"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func (c *Config) finalizeTelemetry() {
	if c.Telemetry.TracerProvider == nil {
		if c.UseEnv {
			c.Telemetry.TracerProvider = otel.GetTracerProvider()
		} else {
			c.Telemetry.TracerProvider = trace.NewNoopTracerProvider()
		}
	}

	if c.Telemetry.MeterProvider == nil {
		if c.UseEnv {
			c.Telemetry.MeterProvider = otel.GetMeterProvider()
		} else {
			c.Telemetry.MeterProvider = noop.NewMeterProvider()
		}
	}

	if c.Telemetry.Logger == nil {
		c.Telemetry.Logger = slog.Default()
	}

	c.Telemetry.Attrs = append(
		c.Telemetry.Attrs,
		telemetry.String("node_id", c.NodeID.AsString()),
	)
}
