package integration

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/marshal"
	"github.com/dogmatiq/veracity/internal/integration/internal/journalpb"
)

// JournalStore is the [journal.Store] type used to integration handler state.
type JournalStore = journal.Store[*journalpb.Record]

// Journal is a journal of records for a specific integration handler.
type Journal = journal.Journal[*journalpb.Record]

// NewJournalStore returns a new [JournalStore] that uses s for its underlying
// storage.
func NewJournalStore(s journal.BinaryStore) JournalStore {
	return journal.NewMarshalingStore(
		s,
		marshal.ProtocolBuffers[*journalpb.Record]{},
	)
}

// JournalName returns the name of the journal that contains the state
// of the integration handler with the given key.
func JournalName(key *uuidpb.UUID) string {
	return "integration:" + key.AsString()
}
