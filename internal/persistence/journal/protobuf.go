package journal

import (
	"context"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// PB is a journal that persists protocol-buffers-based journal records to an
// underlying binary journal.
type PB[R proto.Message] struct {
	Journal BinaryJournal
}

func (j *PB[R]) Read(ctx context.Context, v uint32) (R, bool, error) {
	var r R

	data, ok, err := j.Journal.Read(ctx, v)
	if !ok || err != nil {
		return r, ok, err
	}

	r = reflect.New(
		reflect.TypeOf(r).Elem(),
	).Interface().(R)

	return r, true, proto.Unmarshal(data, r)
}

func (j *PB[R]) Write(ctx context.Context, v uint32, r R) (bool, error) {
	data, err := proto.Marshal(r)
	if err != nil {
		return false, err
	}

	return j.Journal.Write(ctx, v, data)
}
