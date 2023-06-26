package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// A RangeFunc is a function used to range over the records in a [Journal].
type RangeFunc[T proto.Message] func(context.Context, journal.Position, T) (ok bool, err error)

// Get returns the record at the given position.
func Get[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	pos journal.Position,
) (T, bool, error) {
	data, ok, err := j.Get(ctx, pos)
	if !ok || err != nil {
		return nil, ok, err
	}

	rec, err := typedproto.Unmarshal[T](data)
	if err != nil {
		return nil, false, fmt.Errorf("unable to unmarshal record: %w", err)
	}

	return rec, true, nil
}

// Range invokes fn for each record in the journal, in order, beginning at the
// given position.
func Range[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin journal.Position,
	fn RangeFunc[T],
) error {
	return j.Range(
		ctx,
		begin,
		func(ctx context.Context, pos journal.Position, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, pos, rec)
		},
	)
}

// RangeAll invokes fn for each record in the journal, in order.
func RangeAll[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	fn RangeFunc[T],
) error {
	return j.RangeAll(
		ctx,
		func(ctx context.Context, pos journal.Position, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, pos, rec)
		},
	)
}

// Append adds a record to the journal.
func Append[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	end journal.Position,
	rec T,
) error {
	data, err := typedproto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("unable to marshal record: %w", err)
	}

	return j.Append(ctx, end, data)
}

// LatestRecord returns the latest record in the journal.
func LatestRecord[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
) (journal.Position, T, bool, error) {
	for {
		_, end, err := j.Bounds(ctx)
		if end == 0 || err != nil {
			return 0, nil, false, err
		}

		rec, ok, err := Get[T, S](ctx, j, end)
		if err != nil {
			return 0, nil, false, err
		}

		if ok {
			return end, rec, true, nil
		}

		// We didn't find the record, assuming the journal is not corrupted,
		// that means that it was truncated after the call to Bounds() but
		// before the call to Get().
	}
}
