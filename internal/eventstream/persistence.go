package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/marshaler"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb"
)

// newJournalStore returns a new [journal.Store] that uses s for its underlying
// storage.
func newJournalStore(s journal.BinaryStore) journal.Store[*journalpb.Record] {
	return journal.NewMarshalingStore(
		s,
		marshaler.NewProto[*journalpb.Record](),
	)
}

// journalName returns the name of the journal that contains the state
// of the event stream with the given ID.
func journalName(streamID *uuidpb.UUID) string {
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
