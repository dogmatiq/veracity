package protojournal

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
)

// Get returns the record at the given position.
func Get[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	pos journal.Position,
) (Record, error) {
	data, err := j.Get(ctx, pos)
	if err != nil {
		return nil, err
	}

	rec, err := typedproto.Unmarshal[Record](data)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal record: %w", err)
	}

	return rec, nil
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
		begin, end, err := j.Bounds(ctx)
		if begin == end || err != nil {
			return 0, nil, false, err
		}

		pos := end - 1
		rec, err := Get[Record](ctx, j, pos)

		if !errors.Is(err, journal.ErrNotFound) {
			return pos, rec, true, err
		}

		// We didn't find the record, assuming the journal is not corrupted,
		// that means that it was truncated after the call to Bounds() but
		// before the call to Get(), so we re-read the bounds and try again.
	}
}
