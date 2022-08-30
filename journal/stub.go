package journal

import (
	"context"
)

// Stub is a test implementation of the Journal[E] interface.
type Stub[E any] struct {
	Journal[E]

	ReadFunc  func(ctx context.Context, offset uint64) ([]E, uint64, error)
	WriteFunc func(ctx context.Context, offset uint64, entry E) error
}

func (j *Stub[E]) Read(ctx context.Context, offset uint64) (entries []E, next uint64, err error) {
	if j.ReadFunc != nil {
		return j.ReadFunc(ctx, offset)
	}

	if j.Journal != nil {
		return j.Journal.Read(ctx, offset)
	}

	return nil, offset, nil
}

func (j *Stub[E]) Write(ctx context.Context, offset uint64, entry E) (err error) {
	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, offset, entry)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, offset, entry)
	}

	return nil
}
