package integration_test

import (
	"sync/atomic"
	"time"

	. "github.com/dogmatiq/enginekit/enginetest/stubs"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

func newPacker() *envelopepb.Packer {
	var counter atomic.Uint64
	return &envelopepb.Packer{
		Application: identitypb.New("<app>", uuidpb.Generate()),
		Marshaler:   Marshaler,
		Now: func() time.Time {
			return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		},
		GenerateID: func() *uuidpb.UUID {
			return deterministicUUID(counter.Add(1))
		},
	}
}

func deterministicUUID(counter uint64) *uuidpb.UUID {
	var data [16]byte
	data[6] = (data[6] & 0x0f) | 0x40 // Version 4
	data[8] = (data[8] & 0x3f) | 0x80 // Variant is 10 (RFC 4122)

	id := uuidpb.FromByteArray(data)
	id.Lower |= counter

	return id
}
