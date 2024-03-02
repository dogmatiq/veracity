package telemetry

import (
	"fmt"
	"log/slog"
	"math"
	"reflect"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/constraints"
)

// Attr is a telemetry attribute.
type Attr struct {
	typ attrType
	key string
	str string
	num uint64
	sli any
}

// String returns a string attribute.
func String[T ~string](k string, v T) Attr {
	return Attr{
		typ: attrTypeString,
		key: k,
		str: string(v),
	}
}

// Stringer returns a string attribute. The value is the result of calling
// v.String().
func Stringer(k string, v fmt.Stringer) Attr {
	return String(k, v.String())
}

// Type returns a string attribute set to the name of T.
func Type[T any](k string, v T) Attr {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return String(k, t.String())
}

// Bool returns a boolean attribute.
func Bool[T ~bool](k string, v T) Attr {
	var n uint64
	if v {
		n = 1
	}

	return Attr{
		typ: attrTypeBool,
		key: k,
		num: n,
	}
}

// Int returns an int64 attribute.
func Int[T constraints.Integer](k string, v T) Attr {
	return Attr{
		typ: attrTypeInt64,
		key: k,
		num: uint64(v),
	}
}

// If conditionally includes an attribute.
func If(cond bool, attr Attr) Attr {
	if cond {
		return attr
	}
	return Attr{}
}

// Float returns a float64 attribute.
func Float[T constraints.Float](k string, v T) Attr {
	return Attr{
		typ: attrTypeFloat64,
		key: k,
		num: math.Float64bits(float64(v)),
	}
}

func (a Attr) otel() (attribute.KeyValue, bool) {
	switch a.typ {
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

func (a Attr) slog() (slog.Attr, bool) {
	switch a.typ {
	case attrTypeNone:
		return slog.Attr{}, false
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

type attrType uint8

const (
	attrTypeNone attrType = iota
	attrTypeString
	attrTypeBool
	attrTypeInt64
	attrTypeFloat64
)
