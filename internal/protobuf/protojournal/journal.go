package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// A RangeFunc is a function used to range over the records in a [Journal].
type RangeFunc[T proto.Message] func(ctx context.Context, ver uint64, rec T) (ok bool, err error)

// Get returns the record written to produce the given version of the
// journal.
func Get[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	ver uint64,
) (T, bool, error) {
	data, ok, err := j.Get(ctx, ver)
	if !ok || err != nil {
		return nil, ok, err
	}

	rec, err := typedproto.Unmarshal[T](data)
	if err != nil {
		return nil, false, fmt.Errorf("unable to unmarshal record: %w", err)
	}

	return rec, true, nil
}

// Range invokes fn for each record in the journal, beginning at the given
// version, in order.
func Range[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	ver uint64,
	fn RangeFunc[T],
) error {
	return j.Range(
		ctx,
		ver,
		func(ctx context.Context, ver uint64, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, ver, rec)
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
		func(ctx context.Context, ver uint64, data []byte) (bool, error) {
			rec, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, ver, rec)
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
	ver uint64,
	rec T,
) (bool, error) {
	data, err := typedproto.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("unable to marshal record: %w", err)
	}
	return j.Append(ctx, ver, data)
}
