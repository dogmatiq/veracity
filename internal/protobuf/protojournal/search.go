package protojournal

import (
	"context"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// CompareFunc is a function that compares a record to some datum.
//
// If the record is less than the datum, it returns a negative value. If the
// record is greater than the datum, it returns a positive value. If the record
// is equal to the datum, it returns zero.
type CompareFunc[T proto.Message] func(context.Context, T) (int, error)

// Search performs a binary search of the journal to find the position of
// the record for which cmp() returns zero.
func Search[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin, end journal.Position,
	cmp CompareFunc[T],
) (journal.Position, T, bool, error) {
	for begin < end {
		median := (begin >> 1) + (end >> 1)

		rec, ok, err := Get[T](ctx, j, median)
		if err != nil {
			return 0, nil, false, err
		}
		if !ok {
			break
		}

		result, err := cmp(ctx, rec)
		if err != nil {
			return 0, nil, false, err
		}

		if result < 0 {
			end = median
		} else if result > 0 {
			begin = median + 1
		} else {
			return median, rec, true, nil
		}
	}

	return 0, nil, false, nil
}

// SearchThenRange ranges over records in the journal, beginning with the record
// for which cmp() returns 0.
func SearchThenRange[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin, end journal.Position,
	cmp CompareFunc[T],
	fn RangeFunc[T],
) error {
	pos, rec, ok, err := Search[T, S](ctx, j, begin, end, cmp)
	if !ok || err != nil {
		return err
	}

	ok, err = fn(ctx, pos, rec)
	if !ok || err != nil {
		return err
	}

	return Range(ctx, j, pos+1, fn)
}
