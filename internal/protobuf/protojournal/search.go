package protojournal

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

type (
	// CompareFunc is a function that compares a record to some datum.
	//
	// If the record is less than the datum, cmp is negative. If the record is
	// greater than the datum, cmp is positive. Otherwise, the record is
	// considered equal to the datum.
	CompareFunc[Record proto.Message] func(
		ctx context.Context,
		pos journal.Position,
		rec Record,
	) (cmp int, err error)

	// ScanFunc is a predicate function that produces a value of type T from a
	// record.
	ScanFunc[T any, Record proto.Message] func(
		ctx context.Context,
		pos journal.Position,
		rec Record,
	) (value T, ok bool, err error)
)

// Search performs a binary search of the journal to find the position of
// the record for which cmp() returns zero.
func Search[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin, end journal.Position,
	cmp CompareFunc[Record],
) (journal.Position, Record, bool, error) {
	for begin < end {
		pos := (begin >> 1) + (end >> 1)

		rec, err := Get[Record](ctx, j, pos)
		if errors.Is(err, journal.ErrNotFound) {
			break
		} else if err != nil {
			return 0, nil, false, err
		}

		result, err := cmp(ctx, pos, rec)
		if err != nil {
			return 0, nil, false, err
		}

		if result < 0 {
			end = pos
		} else if result > 0 {
			begin = pos + 1
		} else {
			return pos, rec, true, nil
		}
	}

	return 0, nil, false, nil
}
