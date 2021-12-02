package index

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// Index is an interface for persisting the index.
//
// It is a simple binary key/value store.
type Index interface {
	// Set sets the value associated with k to v.
	//
	// There is no distinction between empty and non-existent values. Setting a
	// key to an empty value effectively removes that key.
	Set(ctx context.Context, k, v []byte) error

	// Get returns the value associated with k.
	//
	// It returns an empty value if k is not set.
	Get(ctx context.Context, k []byte) (v []byte, err error)
}

// setProto sets the value of k to v's binary representation.
func setProto(ctx context.Context, i Index, k []byte, v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	return i.Set(ctx, k, data)
}

// getProto reads the value of k into v.
func getProto(ctx context.Context, i Index, k []byte, v proto.Message) error {
	data, err := i.Get(ctx, k)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, v)
}

// formatKey formats a string for use as a key.
func formatKey(f string, v ...interface{}) []byte {
	return []byte(fmt.Sprintf(f, v...))
}
