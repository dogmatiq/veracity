package journal

import (
	"context"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// NewProtoStore returns a new journal store that contains journals with records
// that a protocol buffers messages of type R.
func NewProtoStore[R proto.Message](s BinaryStore) Store[R] {
	return &protoStore[R]{s}
}

// protoStore is a journal store that contains journals with records that a
// protocol buffers messages of type R.
type protoStore[R proto.Message] struct {
	BinaryStore
}

func (s *protoStore[R]) Open(ctx context.Context, path ...string) (Journal[R], error) {
	j, err := s.BinaryStore.Open(ctx, path...)
	if err != nil {
		return nil, err
	}

	return &protoJournal[R]{j}, nil
}

// protoJournal is a journal that stores records as protocol buffers messages of
// type R.
type protoJournal[R proto.Message] struct {
	BinaryJournal
}

func (j *protoJournal[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	data, ok, err := j.BinaryJournal.Read(ctx, ver)
	if !ok || err != nil {
		var zero R
		return zero, ok, err
	}

	rec, err := j.unmarshal(ver, data)
	return rec, true, err
}

func (j *protoJournal[R]) ReadOldest(ctx context.Context) (uint64, R, bool, error) {
	ver, data, ok, err := j.BinaryJournal.ReadOldest(ctx)
	if !ok || err != nil {
		var zero R
		return 0, zero, ok, err
	}

	rec, err := j.unmarshal(ver, data)
	return ver, rec, true, err
}

func (j *protoJournal[R]) Write(ctx context.Context, ver uint64, rec R) (bool, error) {
	data, err := proto.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("unable to marshal journal record: %w", err)
	}

	return j.BinaryJournal.Write(ctx, ver, data)
}

func (j *protoJournal[R]) unmarshal(ver uint64, data []byte) (_ R, _ error) {
	var rec R

	v := reflect.ValueOf(rec)
	v.Set(
		reflect.New(
			v.Type().Elem(),
		),
	)

	if err := proto.Unmarshal(data, rec); err != nil {
		var zero R
		return zero, fmt.Errorf("unable to unmarshal journal record at version %d: %w", ver, err)
	}

	return rec, nil
}
