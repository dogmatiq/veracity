package veracity

import (
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// WithNodeID is an [EngineOption] that sets the node ID of the engine.
func WithNodeID(id uuid.UUID) EngineOption {
	if id == uuid.Nil {
		panic("node ID must not be the nil UUID")
	}

	return func(cfg *engineConfig) {
		cfg.NodeID = id
	}
}

// WithListenAddress is an [EngineOption] that sets the network address for the
// engine's internal gRPC server.
func WithListenAddress(addr string) EngineOption {
	return func(cfg *engineConfig) {
		cfg.GRPC.ListenAddress = addr
	}
}

// WithAdvertiseAddresses is an [EngineOption] that sets the network addresses
// at which the engine's internal gRPC server can be reached by other nodes in
// the cluster.
func WithAdvertiseAddresses(addresses ...string) EngineOption {
	return func(cfg *engineConfig) {
		cfg.GRPC.AdvertiseAddresses = addresses
	}
}

// WithGRPCDialOptions is an [EngineOption] that sets the gRPC options to use
// when connecting to other nodes in the cluster.
func WithGRPCDialOptions(options ...grpc.DialOption) EngineOption {
	return func(cfg *engineConfig) {
		cfg.GRPC.DialOptions = append(cfg.GRPC.DialOptions, options...)
	}
}

// WithGRPCServerOptions is an [EngineOption] that sets
// the gRPC options to use when starting the internal gRPC server.
func WithGRPCServerOptions(options ...grpc.ServerOption) EngineOption {
	return func(cfg *engineConfig) {
		cfg.GRPC.ServerOptions = append(cfg.GRPC.ServerOptions, options...)
	}
}
