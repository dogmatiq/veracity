package protokv

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/veracity/persistence/kv"
	"google.golang.org/protobuf/proto"
)

// Get returns the value associated with k.
func Get[V proto.Message](
	ctx context.Context,
	ks kv.Keyspace,
	k []byte,
) (V, bool, error) {
	var v V

	data, err := ks.Get(ctx, k)
	if err != nil || len(data) == 0 {
		return v, false, err
	}

	t := reflect.TypeOf(v).Elem()
	v = reflect.New(t).Interface().(V)

	if err := proto.Unmarshal(data, v); err != nil {
		return v, false, fmt.Errorf("unable to unmarshal value: %w", err)
	}

	return v, true, nil
}

// Set associates a value with k.
func Set[V proto.Message](
	ctx context.Context,
	ks kv.Keyspace,
	k []byte,
	v V,
) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return fmt.Errorf("unable to marshal value: %w", err)
	}
	return ks.Set(ctx, k, data)
}

// RangeAll invokes fn for each key in ks.
//
// If fn returns false, ranging stops RangeAll() returns immediately.
//
// The order and read isolation is undefined.
func RangeAll[V proto.Message](
	ctx context.Context,
	ks kv.Keyspace,
	fn func(context.Context, []byte, V) (bool, error),
) error {
	return ks.RangeAll(
		ctx,
		func(ctx context.Context, k, data []byte) (bool, error) {
			var v V
			t := reflect.TypeOf(v).Elem()
			v = reflect.New(t).Interface().(V)

			if err := proto.Unmarshal(data, v); err != nil {
				return false, fmt.Errorf("unable to unmarshal value: %w", err)
			}

			return fn(ctx, k, v)
		},
	)
}
