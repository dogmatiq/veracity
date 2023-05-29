package telemetry

import (
	"fmt"
	"math"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slog"
)

// Attr is a telemetry attribute.
type Attr struct {
	flags attrFlags
	key   string
	str   string
	num   uint64
	sli   any
}

// String returns a string attribute.
func String(k, v string) Attr {
	return Attr{
		flags: attrFlags(attrTypeString),
		key:   k,
		str:   v,
	}
}

// Stringer returns a string attribute. The value is the result of calling
// v.String().
func Stringer(k string, v fmt.Stringer) Attr {
	return String(k, v.String())
}

// Bool returns a boolean attribute.
func Bool(k string, v bool) Attr {
	var n uint64
	if v {
		n = 1
	}

	return Attr{
		flags: attrFlags(attrTypeBool),
		key:   k,
		num:   n,
	}
}

// Int returns an int64 attribute.
func Int[T constraints.Integer](k string, v T) Attr {
	return Attr{
		flags: attrFlags(attrTypeInt64),
		key:   k,
		num:   uint64(v),
	}
}

// Float returns a float64 attribute.
func Float[T constraints.Float](k string, v T) Attr {
	return Attr{
		flags: attrFlags(attrTypeFloat64),
		key:   k,
		num:   math.Float64bits(float64(v)),
	}
}

func (a Attr) otel() (attribute.KeyValue, bool) {
	switch a.flags.Type() {
	case attrTypeNone:
		return attribute.KeyValue{}, false
	case attrTypeString:
		return attribute.String(a.key, a.str), true
	case attrTypeBool:
		return attribute.Bool(a.key, a.num != 0), true
	case attrTypeInt64:
		return attribute.Int64(a.key, int64(a.num)), true
	case attrTypeFloat64:
		return attribute.Float64(a.key, math.Float64frombits(a.num)), true
	default:
		panic("unknown attribute type")
	}
}

func (a Attr) slog() (any, bool) {
	switch a.flags.Type() {
	case attrTypeNone:
		return nil, false
	case attrTypeString:
		return slog.String(a.key, a.str), true
	case attrTypeBool:
		return slog.Bool(a.key, a.num != 0), true
	case attrTypeInt64:
		return slog.Int64(a.key, int64(a.num)), true
	case attrTypeFloat64:
		return slog.Float64(a.key, math.Float64frombits(a.num)), true
	default:
		panic("unknown attribute type")
	}
}

// attrSet is a set of attributes, resolved to OpenTelemetry and slog formats.
type attrSet struct {
	Namespace string
	Attrs     []Attr

	otel            []attribute.KeyValue
	slogQualified   []any
	slogUnqualified []any
}

func (s *attrSet) ForOpenTelemetryX() []attribute.KeyValue {
	return s.otel
}

func (s *attrSet) ForTracer() trace.TracerOption {
	return trace.WithInstrumentationAttributes(s.otel...)
}

func (s *attrSet) ForSpan() trace.SpanStartEventOption {
	return trace.WithAttributes(s.otel...)
}

func (s *attrSet) ForMeter() metric.MeterOption {
	return metric.WithInstrumentationAttributes(s.otel...)
}

func (s *attrSet) ForMeasurement() metric.MeasurementOption {
	return metric.WithAttributes(s.otel...)
}

func (s *attrSet) ForLogger() []any {
	return append(
		s.slogQualified,
		slog.Group(s.Namespace, s.slogUnqualified...),
	)
}

func (s *attrSet) resolve() {
	if s.Attrs == nil {
		return
	}

	for _, attr := range s.Attrs {
		if a, ok := attr.otel(); ok {
			if attr.flags.IsQualified() {
				s.otel = append(s.otel, a)
			} else {
				s.otel = append(s.otel, attribute.KeyValue{
					Key:   "io.dogmatiq.veracity." + attribute.Key(s.Namespace) + "." + a.Key,
					Value: a.Value,
				})
			}
		}

		if a, ok := attr.slog(); ok {
			if attr.flags.IsQualified() {
				s.slogQualified = append(s.slogQualified, a)
			} else {
				s.slogUnqualified = append(s.slogUnqualified, a)
			}
		}
	}

	s.Attrs = nil
}

type (
	attrFlags uint8
	attrType  uint8
)

const (
	attrFlagTypeMask        attrFlags = 0b0_000_1111
	attrFlagIsQualifiedMask attrFlags = 0b1_000_0000
	attrFlagReservedMask    attrFlags = 0b0_111_0000

	attrTypeNone attrType = iota
	attrTypeString
	attrTypeBool
	attrTypeInt64
	attrTypeFloat64
)

// IsQualified returns true if the attribute key is fully qualified.
//
// Fully qualified attribute keys are NOT prefixed with the subsystem name.
func (f attrFlags) IsQualified() bool {
	return (f & attrFlagIsQualifiedMask) != 0
}

// Type returns the attribute type.
func (f attrFlags) Type() attrType {
	return attrType(f & attrFlagTypeMask)
}
