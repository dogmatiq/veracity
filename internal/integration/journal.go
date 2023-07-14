package integration

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// JournalName returns the name of the journal that contains the state
// of the integration handler with the given key.
func JournalName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString()
}
