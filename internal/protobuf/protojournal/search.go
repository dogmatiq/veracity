package protojournal

import (
	"context"

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

		rec, ok, err := Get[Record](ctx, j, pos)
		if err != nil {
			return 0, nil, false, err
		}
		if !ok {
			break
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

// Scan finds a value within the journal by scanning all records
// beginning with the record at the given position.
func Scan[
	T any,
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin journal.Position,
	scan ScanFunc[T, Record],
) (value T, ok bool, err error) {
	err = Range(
		ctx,
		j,
		begin,
		func(ctx context.Context, pos journal.Position, rec Record) (bool, error) {
			value, ok, err = scan(ctx, pos, rec)
			return !ok, err
		},
	)
	return value, ok, err
}

// ScanAll finds a value within the journal by scanning all records.
func ScanAll[
	T any,
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	scan ScanFunc[T, Record],
) (value T, ok bool, err error) {
	err = RangeAll(
		ctx,
		j,
		func(ctx context.Context, pos journal.Position, rec Record) (bool, error) {
			value, ok, err = scan(ctx, pos, rec)
			return !ok, err
		},
	)
	return value, ok, err
}

// ScanFromSearchResult finds a value within the journal by scanning all records
// beginning with the for which cmp() returns true.
func ScanFromSearchResult[
	T any,
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	begin, end journal.Position,
	cmp CompareFunc[Record],
	scan ScanFunc[T, Record],
) (value T, ok bool, err error) {
	pos, rec, ok, err := Search(ctx, j, begin, end, cmp)
	if !ok || err != nil {
		return value, false, err
	}

	value, ok, err = scan(ctx, pos, rec)
	if ok || err != nil {
		return value, ok, err
	}

	return Scan(ctx, j, pos+1, scan)
}