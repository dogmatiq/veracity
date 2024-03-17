package cluster

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/persistencekit/marshaler"
	"github.com/dogmatiq/veracity/internal/cluster/internal/registrypb"
)

// newKVStore returns a new [kv.Store] that uses s for its underlying storage.
func newKVStore(s kv.BinaryStore) kv.Store[*uuidpb.UUID, *registrypb.Registration] {
	return kv.NewMarshalingStore(
		s,
		marshaler.NewProto[*uuidpb.UUID](),
		marshaler.NewProto[*registrypb.Registration](),
	)
}

// registryKeyspace is the name of the keyspace that contains registry data.
const registryKeyspace = "cluster.registry"
