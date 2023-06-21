package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// A RangeFunc is a function used to range over the records in a [Journal].
type RangeFunc[T proto.Message] func(context.Context, journal.Offset, T) (ok bool, err error)

// Get returns the record at the given offset.
func Get[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	off journal.Offset,
) (T, bool, error) {
	data, ok, err := j.Get(ctx, off)
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
// given offset.
func Range[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin journal.Offset,
	fn RangeFunc[T],
) error {
	return j.Range(
		ctx,
		begin,
		func(ctx context.Context, off journal.Offset, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, off, rec)
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
		func(ctx context.Context, off journal.Offset, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, off, rec)
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
	end journal.Offset,
	rec T,
) error {
	data, err := typedproto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("unable to marshal record: %w", err)
	}

	return j.Append(ctx, end, data)
}
