package index

import (
	"bytes"
	"context"

	"google.golang.org/protobuf/proto"
)

// Index is an interface for storing indexes that allow the engine to
// efficiently reference information contained in the application's journal.
//
// The index can always be fully recovered from an in-tact journal, however
// rebuilding the index may take a long time.
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
func setProto(ctx context.Context, index Index, k []byte, v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	return index.Set(ctx, k, data)
}

// getProto reads the value of k into v.
func getProto(ctx context.Context, index Index, k []byte, v proto.Message) error {
	data, err := index.Get(ctx, k)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, v)
}

// makeKey returns a key made from slash-separated parts.
func makeKey(parts ...string) []byte {
	var buf bytes.Buffer

	for i, p := range parts {
		if i > 0 {
			buf.WriteByte('/')
		}

		buf.WriteString(p)
	}

	return buf.Bytes()
}
