package protojournal

import (
	"context"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// Search performs a binary search of the journal to find the position of
// the record for which cmp() returns zero.
func Search[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin, end journal.Position,
	cmp func(context.Context, T) (int, error),
) (journal.Position, bool, error) {
	for begin < end {
		median := (begin >> 1) + (end >> 1)

		rec, ok, err := Get[T](ctx, j, median)
		if err != nil {
			return 0, false, err
		}
		if !ok {
			break
		}

		result, err := cmp(ctx, rec)
		if err != nil {
			return 0, false, err
		}

		if result < 0 {
			end = median
		} else if result > 0 {
			begin = median + 1
		} else {
			return median, true, nil
		}
	}

	return 0, false, nil
}
