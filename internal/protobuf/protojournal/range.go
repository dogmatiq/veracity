package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// A RangeFunc is called by [Range] and [RangeAll] for each record in a
// [journal.Journal].
type RangeFunc[Record proto.Message] func(
	ctx context.Context,
	pos journal.Position,
	rec Record,
) (ok bool, err error)

// Range invokes fn for each record in the journal, in order, beginning at the
// given position.
func Range[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin journal.Position,
	fn RangeFunc[Record],
) error {
	return j.Range(
		ctx,
		begin,
		func(
			ctx context.Context,
			pos journal.Position,
			data []byte,
		) (bool, error) {
			rec, err := typedproto.Unmarshal[Record](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, pos, rec)
		},
	)
}

// RangeAll invokes fn for each record in the journal, in order.
func RangeAll[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	fn RangeFunc[Record],
) error {
	return j.RangeAll(
		ctx,
		func(
			ctx context.Context,
			pos journal.Position,
			data []byte,
		) (bool, error) {
			rec, err := typedproto.Unmarshal[Record](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}
			return fn(ctx, pos, rec)
		},
	)
}
