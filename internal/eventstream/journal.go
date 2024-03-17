package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/marshal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb"
)

// JournalStore is the [journal.Store] type used to store event stream state.
// Each event stream has its own journal.
type JournalStore = journal.Store[*journalpb.Record]

// Journal is a journal of event stream records for a single event stream.
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
// of the event stream with the given ID.
func JournalName(streamID *uuidpb.UUID) string {
	return "eventstream:" + streamID.AsString()
}

// searchByOffset returns a compare function that searches for the journal
// record that contains the event with the given offset.
func searchByOffset(off Offset) journal.CompareFunc[*journalpb.Record] {
	return func(
		ctx context.Context,
		pos journal.Position,
		rec *journalpb.Record,
	) (int, error) {
		if rec.StreamOffsetBefore > uint64(off) {
			return +1, nil
		} else if rec.StreamOffsetAfter <= uint64(off) {
			return -1, nil
		}
		return 0, nil
	}
}
