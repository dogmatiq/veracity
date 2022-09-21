package protojournal

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// Get reads a record from j and unmarshals it into rec.
func Get(
	ctx context.Context,
	j journal.Journal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, ok, err := j.Get(ctx, ver)
	if !ok || err != nil {
		return false, err
	}

	return true, proto.Unmarshal(data, rec)
}

// RangeAll invokes fn for each record in j, in order.
func RangeAll[R proto.Message](
	ctx context.Context,
	j journal.Journal,
	fn func(context.Context, uint64, R) (bool, error),
) error {
	var rec R
	rec = reflect.New(
		reflect.TypeOf(rec).Elem(),
	).Interface().(R)

	return j.RangeAll(
		ctx,
		func(ctx context.Context, ver uint64, data []byte) (bool, error) {
			if err := proto.Unmarshal(data, rec); err != nil {
				return false, fmt.Errorf("unable to unmarshal record: %w", err)
			}

			return fn(ctx, ver, rec)
		},
	)
}

// Append marshals rec to its binary representation and appends it to j.
func Append(
	ctx context.Context,
	j journal.Journal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, err := proto.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("unable to marshal journal record: %w", err)
	}

	return j.Append(ctx, ver, data)
}
