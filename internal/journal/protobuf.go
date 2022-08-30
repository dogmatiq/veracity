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

func (j *PB[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	var rec R

	data, ok, err := j.Journal.Read(ctx, ver)
	if !ok || err != nil {
		return rec, ok, err
	}

	rec = reflect.New(
		reflect.TypeOf(rec).Elem(),
	).Interface().(R)

	return rec, true, proto.Unmarshal(data, rec)
}

func (j *PB[R]) Write(ctx context.Context, ver uint64, rec R) error {
	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	return j.Journal.Write(ctx, ver, data)
}
