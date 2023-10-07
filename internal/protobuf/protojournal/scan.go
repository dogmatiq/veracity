package protojournal

import (
	"context"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
)

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
