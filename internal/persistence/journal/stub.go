package journal

import "context"

// Stub is a test implementation of the Journal[R] interface.
type Stub[R any] struct {
	Journal[R]

	ReadFunc  func(ctx context.Context, v uint32) (R, bool, error)
	WriteFunc func(ctx context.Context, v uint32, r R) (bool, error)
}

func (j *Stub[R]) Read(ctx context.Context, v uint32) (R, bool, error) {
	if j.ReadFunc != nil {
		return j.ReadFunc(ctx, v)
	}

	if j.Journal != nil {
		return j.Journal.Read(ctx, v)
	}

	var zero R
	return zero, false, nil
}

func (j *Stub[R]) Write(ctx context.Context, v uint32, r R) (bool, error) {
	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, v, r)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, v, r)
	}

	return false, nil
}
