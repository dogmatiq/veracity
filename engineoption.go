package veracity

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/persistencekit/set"
	"github.com/dogmatiq/veracity/internal/engineconfig"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// FerriteRegistry is a registry of the environment variables used by Veracity.
//
// It can be used with the [ferrite] package.
var FerriteRegistry = engineconfig.FerriteRegistry

// An EngineOption configures the behavior of an [Engine].
type EngineOption func(*engineconfig.Config)

// WithOptionsFromEnvironment is an engine option that configures the engine
// using options specified via environment variables.
//
// Any explicit options passed to [New] take precedence over options from the
// environment.
func WithOptionsFromEnvironment() EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.UseEnv = true
	}
}

// WithNodeID is an [EngineOption] that sets the node ID of the engine.
func WithNodeID(id *uuidpb.UUID) EngineOption {
	if err := id.Validate(); err != nil {
		panic(err)
	}

	return func(cfg *engineconfig.Config) {
		cfg.NodeID = id
	}
}

// WithTracerProvider is an [EngineOption] that sets the OpenTelemetry tracer
// provider used by the engine.
func WithTracerProvider(p trace.TracerProvider) EngineOption {
	if p == nil {
		panic("tracer provider must not be nil")
	}

	return func(cfg *engineconfig.Config) {
		cfg.Telemetry.TracerProvider = p
	}
}

// WithMetricProvider is an [EngineOption] that sets the OpenTelemetry meter
// provider used by the engine.
func WithMetricProvider(p metric.MeterProvider) EngineOption {
	if p == nil {
		panic("metric provider must not be nil")
	}

	return func(cfg *engineconfig.Config) {
		cfg.Telemetry.MeterProvider = p
	}
}

// WithLoggerProvider is an [EngineOption] that sets the OpenTelemetry logger
// provider used by the engine.
func WithLoggerProvider(p log.LoggerProvider) EngineOption {
	if p == nil {
		panic("logger provider must not be nil")
	}

	return func(cfg *engineconfig.Config) {
		cfg.Telemetry.LoggerProvider = p
	}
}

// WithJournalStore is an [EngineOption] that sets the journal store used by the
// engine.
func WithJournalStore(s journal.BinaryStore) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.Persistence.Journals = s
	}
}

// WithKeyValueStore is an [EngineOption] that sets the key/value store used by
// the engine.
func WithKeyValueStore(s kv.BinaryStore) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.Persistence.Keyspaces = s
	}
}

// WithSetStore is an [EngineOption] that sets the set store used by the engine.
func WithSetStore(s set.BinaryStore) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.Persistence.Sets = s
	}
}

// WithGRPCListenAddress is an [EngineOption] that sets the network address for
// the engine's internal gRPC server.
func WithGRPCListenAddress(addr string) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.GRPC.ListenAddress = addr
	}
}

// WithGRPCAdvertiseAddresses is an [EngineOption] that sets the network
// addresses at which the engine's internal gRPC server can be reached by other
// nodes in the cluster.
func WithGRPCAdvertiseAddresses(addresses ...string) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.GRPC.AdvertiseAddresses = addresses
	}
}

// WithGRPCDialOptions is an [EngineOption] that sets the gRPC options to use
// when connecting to other nodes in the cluster.
func WithGRPCDialOptions(options ...grpc.DialOption) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.GRPC.DialOptions = append(cfg.GRPC.DialOptions, options...)
	}
}

// WithGRPCServerOptions is an [EngineOption] that sets
// the gRPC options to use when starting the internal gRPC server.
func WithGRPCServerOptions(options ...grpc.ServerOption) EngineOption {
	return func(cfg *engineconfig.Config) {
		cfg.GRPC.ServerOptions = append(cfg.GRPC.ServerOptions, options...)
	}
}
