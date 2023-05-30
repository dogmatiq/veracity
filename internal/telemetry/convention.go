package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	// ReadDirection is an attribute that indicates a read operation.
	ReadDirection = metric.WithAttributeSet(
		attribute.NewSet(
			attribute.String("direction", "read"),
		),
	)

	// WriteDirection is an attribute that indicates a write operation.
	WriteDirection = metric.WithAttributeSet(
		attribute.NewSet(
			attribute.String("direction", "write"),
		),
	)
)
