package journal

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Marshaler is an interface for marshaling Protocol Buffers messages.
type Marshaler interface {
	Marshal(proto.Message) ([]byte, error)
}

// Unmarshaler is an interface for unmarshaling Protocol Buffers messages.
type Unmarshaler interface {
	Unmarshal([]byte, proto.Message) error
}

var (
	// DefaultMarshaler is the default marshaler to use for marshaling Protocol
	// Buffers messages.
	DefaultMarshaler Marshaler = proto.MarshalOptions{}

	// DefaultUnmarshaler is the default unmarshaler to use for unmarshaling
	// Protocol Buffers messages.
	DefaultUnmarshaler Unmarshaler = proto.UnmarshalOptions{}
)

// marshal marshals a protocol buffers message.
func (c *Committer) marshal(m proto.Message) ([]byte, error) {
	marshaler := c.Marshaler
	if marshaler == nil {
		marshaler = DefaultMarshaler
	}

	return marshaler.Marshal(m)
}

// unmarshal unmarshals a protocol buffers message.
func (c *Committer) unmarshal(data []byte, m proto.Message) error {
	unmarshaler := c.Unmarshaler
	if unmarshaler == nil {
		unmarshaler = DefaultUnmarshaler
	}

	return unmarshaler.Unmarshal(data, m)
}

// set sets the value of k to v's binary representation.
func (c *Committer) set(ctx context.Context, k []byte, v proto.Message) error {
	data, err := c.marshal(v)
	if err != nil {
		return err
	}

	return c.Index.Set(ctx, k, data)
}

// get reads the value of k into v.
func (c *Committer) get(ctx context.Context, k []byte, v proto.Message) error {
	data, err := c.Index.Get(ctx, k)
	if err != nil {
		return err
	}

	return c.unmarshal(data, v)
}

// update gets the value associated with k, calls update() then stores the new
// value.
func (c *Committer) update(
	ctx context.Context,
	k []byte,
	v proto.Message,
	update func(),
) error {
	if err := c.get(ctx, k, v); err != nil {
		return err
	}

	update()

	return c.set(ctx, k, v)
}
