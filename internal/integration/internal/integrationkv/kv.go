package integrationkv

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/persistencekit/marshaler"
)

// OpenHandledCommands opens the keyspace that contains the set of handled
// command IDs for the integration handler with the given key.
func OpenHandledCommands(
	ctx context.Context,
	s kv.BinaryStore,
	key *uuidpb.UUID,
) (kv.Keyspace[*uuidpb.UUID, bool], error) {
	store := kv.NewMarshalingStore(s, uuidMarshaler, marshaler.Bool)
	name := HandledCommandsKeyspace(key)
	return store.Open(ctx, name)
}

var uuidMarshaler = marshaler.NewProto[*uuidpb.UUID]()

// HandledCommandsKeyspace returns the name of the keyspace that contains the
// set of handled command IDs for the integration handler with the given key.
func HandledCommandsKeyspace(key *uuidpb.UUID) string {
	return "integration:" + key.AsString() + ":handled-commands"
}
