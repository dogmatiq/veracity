package typedproto

import "google.golang.org/protobuf/proto"

// MessageStruct is a "place-holder constraint" that indicates a type parameter
// should be a generated struct, a pointer to which implements implements
// [proto.Message].
type MessageStruct interface{}

// Message is a constraint for a [proto.Message] implemented by type *S.
//
// Even though Protocol Buffers generates structs that use pointer receivers, S
// is the "raw" struct type, not a pointer to it.
type Message[S MessageStruct] interface {
	proto.Message
	*S
}

// New returns a new zero-valued message of type T.
func New[
	T Message[S],
	S MessageStruct,
]() T {
	var m S
	return &m
}

// Unmarshal unmarshals data into a new message of type T.
func Unmarshal[
	T Message[S],
	S MessageStruct,
](data []byte) (T, error) {
	m := New[T, S]()

	if err := proto.Unmarshal(data, m); err != nil {
		return nil, err
	}

	return m, nil
}

// Marshal marshals m to its wire representation.
//
// This function is provided for symmetry with [Unmarshal], but is otherwise
// functionally an alias for [proto.Marshal].
func Marshal[
	T Message[S],
	S MessageStruct,
](m T) ([]byte, error) {
	return proto.Marshal(m)
}

// Clone returns a deep copy of m.
func Clone[
	T Message[S],
	S MessageStruct,
](m T) T {
	return proto.Clone(m).(T)
}
