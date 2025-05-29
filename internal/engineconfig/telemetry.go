package engineconfig

import (
	"log/slog"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"go.opentelemetry.io/otel"
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

	if c.Telemetry.Logger == nil {
		c.Telemetry.Logger = slog.Default()
	}

	c.Telemetry.Attrs = append(
		c.Telemetry.Attrs,
		telemetry.String("node_id", c.NodeID.AsString()),
	)
}
