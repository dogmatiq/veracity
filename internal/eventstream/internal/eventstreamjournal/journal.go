package eventstreamjournal

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/marshaler"
)

// Open returns a journal that contains the state of the event stream with the
// given ID.
func Open(
	ctx context.Context,
	s journal.BinaryStore,
	streamID *uuidpb.UUID,
) (journal.Journal[*Record], error) {
	store := journal.NewMarshalingStore(s, recordMarshaler)
	name := Name(streamID)
	return store.Open(ctx, name)
}

var recordMarshaler = marshaler.NewProto[*Record]()

// Name returns the name of the journal that contains the state of the event
// stream with the given ID.
func Name(streamID *uuidpb.UUID) string {
	return "eventstream:" + streamID.AsString()
}

// SearchByOffset returns a comparison function that searches for the journal
// record that contains the event at the given offset.
func SearchByOffset(off uint64) journal.CompareFunc[*Record] {
	return func(
		ctx context.Context,
		pos journal.Position,
		rec *Record,
	) (int, error) {
		if rec.StreamOffsetBefore > off {
			return +1, nil
		} else if rec.StreamOffsetAfter <= off {
			return -1, nil
		}
		return 0, nil
	}
}
