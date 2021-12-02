package kvstore

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// Store is an interface for a binary key/value store.
type Store interface {
	Set(ctx context.Context, k, v []byte) error
	Get(ctx context.Context, k []byte) ([]byte, error)
}

// SetProto sets the value of k to v's binary representation.
func SetProto(ctx context.Context, s Store, k []byte, v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	return s.Set(ctx, k, data)
}

// GetProto reads the value of k into v.
func GetProto(ctx context.Context, s Store, k []byte, v proto.Message) error {
	data, err := s.Get(ctx, k)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, v)
}

// Key formats a string for use as a key.
func Key(f string, v ...interface{}) []byte {
	return []byte(fmt.Sprintf(f, v...))
}
