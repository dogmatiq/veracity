package integration

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// handledCommandsKeyspaceName returns the name of the keyspace that contains
// the set of handled command IDs.
func handledCommandsKeyspaceName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString() + ":handled-commands"
}
