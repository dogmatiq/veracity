package uuidpb

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
)

// New returns a new randonly generated UUID.
func New() *UUID {
	id := uuid.New()
	return FromNative(id)
}

// FromNative returns a new UUID from the native Go representation.
func FromNative(id uuid.UUID) *UUID {
	return &UUID{
		Upper: binary.BigEndian.Uint64(id[0:]),
		Lower: binary.BigEndian.Uint64(id[8:]),
	}
}

// ToNative returns the native Go representation of the UUID.
func (x *UUID) ToNative() uuid.UUID {
	var id uuid.UUID
	if x != nil {
		binary.BigEndian.AppendUint64(id[0:0], x.Upper)
		binary.BigEndian.AppendUint64(id[8:8], x.Lower)
	}
	return id
}

// ToBytes returns the UUID as a byte slice.
func (x *UUID) ToBytes() []byte {
	id := x.ToNative()
	return id[:]
}

// ToString returns the UUID as an RFC 4122 string.
func (x *UUID) ToString() string {
	return x.ToNative().String()
}

// Format implements the fmt.Formatter interface, allowing UUIDs to be formatted
// with functions from the fmt package.
func (x *UUID) Format(f fmt.State, verb rune) {
	fmt.Fprintf(
		f,
		fmt.FormatString(f, verb),
		x.ToNative(),
	)
}
