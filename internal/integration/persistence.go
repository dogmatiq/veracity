package integration

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// journalName returns the name of the journal that contains the state
// of the integration handler with the given key.
func journalName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString()
}

// handledCommandsKeyspaceName returns the name of the keyspace that contains
// the set of handled command IDs.
func handledCommandsKeyspaceName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString() + ":handled-commands"
}
