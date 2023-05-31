package veracity

import "github.com/google/uuid"

// WithNodeID is an [EngineOption] and [EngineRunOption] that sets the node ID
// of the engine.
func WithNodeID(id uuid.UUID) interface {
	EngineOption
	EngineRunOption
} {
	if id == uuid.Nil {
		panic("node ID must not be the nil UUID")
	}

	fn := func(e *Engine) {
		e.nodeID = id
	}

	return option{
		engineOption:    fn,
		engineRunOption: fn,
	}
}

// WithListenAddress is an [EngineOption] and [EngineRunOption] that sets the
// network address for the engine's internal gRPC server.
func WithListenAddress(addr string) interface {
	EngineOption
	EngineRunOption
} {
	fn := func(e *Engine) {
		e.listenAddress = addr
	}

	return option{
		engineOption:    fn,
		engineRunOption: fn,
	}
}

// WithAdvertiseAddresses is an [EngineOption] and [EngineRunOption] that sets
// the network addresses at which the engine's internal gRPC server can be
// reached by other nodes in the cluster.
func WithAdvertiseAddresses(addresses ...string) interface {
	EngineOption
	EngineRunOption
} {
	fn := func(e *Engine) {
		e.advertiseAddresses = addresses
	}

	return option{
		engineOption:    fn,
		engineRunOption: fn,
	}
}
