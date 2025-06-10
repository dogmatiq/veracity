package telemetry

import (
	"runtime/debug"
	"slices"

	"go.opentelemetry.io/otel/log"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// Provider provides Recorder instances scoped to particular subsystems.
//
// The zero value of a *Provider is equivalent to a provider configured with
// no-op tracer, meter and logging providers.
type Provider struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	LoggerProvider log.LoggerProvider
	Attrs          []Attr
}

// Recorder records traces, metrics and logs for a particular subsystem.
type Recorder struct {
	tracer trace.Tracer
	meter  metric.Meter
	logger log.Logger

	errorCount              Instrument[int64]
	operationCount          Instrument[int64]
	operationsInFlightCount Instrument[int64]
}

// Recorder returns a new Recorder instance.
func (p *Provider) Recorder(attrs ...Attr) *Recorder {
	const pkg = "github.com/dogmatiq/veracity/"

	var (
		tracerProvider trace.TracerProvider
		meterProvider  metric.MeterProvider
		loggerProvider log.LoggerProvider
	)

	if p != nil {
		tracerProvider = p.TracerProvider
		meterProvider = p.MeterProvider
		loggerProvider = p.LoggerProvider

		attrs = append(
			slices.Clone(p.Attrs),
			attrs...,
		)
	}

	if tracerProvider == nil {
		tracerProvider = nooptrace.NewTracerProvider()
	}

	if meterProvider == nil {
		meterProvider = noopmetric.NewMeterProvider()
	}

	if loggerProvider == nil {
		loggerProvider = nooplog.NewLoggerProvider()
	}

	r := &Recorder{
		tracer: tracerProvider.Tracer(
			pkg,
			tracerVersion,
			trace.WithInstrumentationAttributes(asAttrKeyValues(attrs)...),
		),
		meter: meterProvider.Meter(
			pkg,
			meterVersion,
			metric.WithInstrumentationAttributes(asAttrKeyValues(attrs)...),
		),
		logger: loggerProvider.Logger(
			pkg,
			logVersion,
			log.WithInstrumentationAttributes(asAttrKeyValues(attrs)...),
		),
	}

	r.errorCount = r.Counter("errors", "{error}", "The number of errors that have occurred.")
	r.operationCount = r.Counter("operations", "{operation}", "The number of operations that have been performed.")
	r.operationsInFlightCount = r.UpDownCounter("operations.inflight", "{operation}", "The number of operations that are currently in progress.")

	return r
}

var (
	// tracerVersion is a TracerOption that sets the instrumentation version
	// to the current version of the module.
	tracerVersion trace.TracerOption

	// meterVersion is a MeterOption that sets the instrumentation version to
	// the current version of the module.
	meterVersion metric.MeterOption

	// logVersion is a LoggerOption that sets the instrumentation version to
	// the current version of the module.
	logVersion log.LoggerOption
)

func init() {
	const modulePath = "github.com/dogmatiq/veracity"
	version := "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == modulePath {
				version = dep.Version
				break
			}
		}
	}

	tracerVersion = trace.WithInstrumentationVersion(version)
	meterVersion = metric.WithInstrumentationVersion(version)
	logVersion = log.WithInstrumentationVersion(version)
}
