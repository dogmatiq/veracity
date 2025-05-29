package integrationset

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/marshaler"
	"github.com/dogmatiq/persistencekit/set"
)

// OpenAcceptedCommandsSet opens the set that contains the set command IDs that
// have been accepted for processing by the integration handler with the given
// key.
func OpenAcceptedCommandsSet(
	ctx context.Context,
	s set.BinaryStore,
	key *uuidpb.UUID,
) (set.Set[*uuidpb.UUID], error) {
	store := set.NewMarshalingStore(s, uuidMarshaler)
	name := AcceptedCommandsSet(key)
	return store.Open(ctx, name)
}

var uuidMarshaler = marshaler.NewProto[*uuidpb.UUID]()

// AcceptedCommandsSet returns the name of the keyspace that contains the
// set of handled command IDs for the integration handler with the given key.
func AcceptedCommandsSet(key *uuidpb.UUID) string {
	return "integration:" + key.AsString() + ":accepted-commands"
}
