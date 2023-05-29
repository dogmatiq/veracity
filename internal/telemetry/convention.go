package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	// ReadDirection is an attribute that indicates a read operation.
	ReadDirection = metric.WithAttributes(
		attribute.String("direction", "read"),
	)

	// WriteDirection is an attribute that indicates a write operation.
	WriteDirection = metric.WithAttributes(
		attribute.String("direction", "write"),
	)
)
