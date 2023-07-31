package protokv

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/kv"
	"google.golang.org/protobuf/proto"
)

// A RangeFunc is a function used to range over the key/value pairs in a
// [Keyspace].
type RangeFunc[T proto.Message] func(ctx context.Context, k []byte, v T) (ok bool, err error)

// Get returns the value associated with k.
func Get[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	ks kv.Keyspace,
	k []byte,
) (T, bool, error) {
	data, err := ks.Get(ctx, k)
	if err != nil || len(data) == 0 {
		return nil, false, err
	}

	v, err := typedproto.Unmarshal[T](data)
	if err != nil {
		return nil, false, fmt.Errorf("unable to unmarshal value: %w", err)
	}

	return v, true, nil
}

// Set associates a value with k.
func Set[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	ks kv.Keyspace,
	k []byte,
	v T,
) error {
	data, err := typedproto.Marshal(v)
	if err != nil {
		return fmt.Errorf("unable to marshal value: %w", err)
	}
	return ks.Set(ctx, k, data)
}

// Range invokes fn for each key in ks.
func Range[
	T typedproto.Message[S],
	S typedproto.MessageStruct,
](
	ctx context.Context,
	ks kv.Keyspace,
	fn RangeFunc[T],
) error {
	return ks.Range(
		ctx,
		func(ctx context.Context, k, data []byte) (bool, error) {
			v, err := typedproto.Unmarshal[T](data)
			if err != nil {
				return false, fmt.Errorf("unable to unmarshal value: %w", err)
			}
			return fn(ctx, k, v)
		},
	)
}
