package journal

import (
	"context"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// PB is a journal that persists protocol-buffers-based journal records to an
// underlying binary journal.
type PB[R proto.Message] struct {
	BinaryJournal
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *PB[R]) Read(ctx context.Context, v uint64) (R, bool, error) {
	var r R

	data, ok, err := j.BinaryJournal.Read(ctx, v)
	if !ok || err != nil {
		return r, ok, err
	}

	r = reflect.New(
		reflect.TypeOf(r).Elem(),
	).Interface().(R)

	return r, true, proto.Unmarshal(data, r)
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// If v > current then the behavior is undefined.
func (j *PB[R]) Write(ctx context.Context, v uint64, r R) (bool, error) {
	data, err := proto.Marshal(r)
	if err != nil {
		return false, err
	}

	return j.BinaryJournal.Write(ctx, v, data)
}
