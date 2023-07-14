package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb"
	"github.com/dogmatiq/veracity/internal/protobuf/protojournal"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalName returns the name of the journal that contains the state
// of the event stream with the given ID.
func JournalName(streamID *uuidpb.UUID) string {
	return "eventstream:" + streamID.AsString()
}

// searchByOffset returns a compare function that searches for the journal
// record that contains the event with the given offset.
func searchByOffset(off Offset) protojournal.CompareFunc[*journalpb.Record] {
	return func(
		ctx context.Context,
		pos journal.Position,
		rec *journalpb.Record,
	) (int, error) {
		if rec.StreamOffsetBefore > uint64(off) {
			return +1, nil
		} else if rec.StreamOffsetAfter <= uint64(off) {
			return -1, nil
		} else {
			return 0, nil
		}
	}
}
