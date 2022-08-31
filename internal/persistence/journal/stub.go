package journal

import "context"

// Stub is a test implementation of the Journal[R] interface.
type Stub[R any] struct {
	Journal[R]

	ReadFunc  func(ctx context.Context, ver uint64) (R, bool, error)
	WriteFunc func(ctx context.Context, ver uint64, rec R) (bool, error)
}

func (j *Stub[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	if j.ReadFunc != nil {
		return j.ReadFunc(ctx, ver)
	}

	if j.Journal != nil {
		return j.Journal.Read(ctx, ver)
	}

	var zero R
	return zero, false, nil
}

func (j *Stub[R]) Write(ctx context.Context, ver uint64, rec R) (bool, error) {
	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, ver, rec)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, ver, rec)
	}

	return false, nil
}
