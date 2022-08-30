package occjournal

import "context"

// Stub is a test implementation of the Journal[R] interface.
type Stub[R any] struct {
	Journal[R]

	ReadFunc  func(ctx context.Context, version uint64) ([]R, uint64, error)
	WriteFunc func(ctx context.Context, version uint64, entry R) error
}

func (j *Stub[E]) Read(ctx context.Context, version uint64) (entries []E, next uint64, err error) {
	if j.ReadFunc != nil {
		return j.ReadFunc(ctx, version)
	}

	if j.Journal != nil {
		return j.Journal.Read(ctx, version)
	}

	return nil, version, nil
}

func (j *Stub[E]) Write(ctx context.Context, version uint64, entry E) (err error) {
	if j.WriteFunc != nil {
		return j.WriteFunc(ctx, version, entry)
	}

	if j.Journal != nil {
		return j.Journal.Write(ctx, version, entry)
	}

	return nil
}
