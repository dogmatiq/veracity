package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// Get returns the record at the given position.
func Get[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	pos journal.Position,
) (Record, bool, error) {
	data, ok, err := j.Get(ctx, pos)
	if !ok || err != nil {
		return nil, ok, err
	}

	rec, err := typedproto.Unmarshal[Record](data)
	if err != nil {
		return nil, false, fmt.Errorf("unable to unmarshal record: %w", err)
	}

	return rec, true, nil
}

// GetLatest returns the most recent record in the journal.
func GetLatest[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
) (journal.Position, Record, bool, error) {
	for {
		_, end, err := j.Bounds(ctx)
		if end == 0 || err != nil {
			return 0, nil, false, err
		}

		rec, ok, err := Get[Record](ctx, j, end-1)
		if err != nil {
			return 0, nil, false, err
		}

		if ok {
			return end, rec, true, nil
		}

		// We didn't find the record, assuming the journal is not corrupted,
		// that means that it was truncated after the call to Bounds() but
		// before the call to Get(), so we re-read the bounds and try again.
	}
}
