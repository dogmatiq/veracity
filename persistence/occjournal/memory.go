package occjournal

import (
	"context"
	"sync"
)

// InMemory is an in-memory journal.
type InMemory[R any] struct {
	m       sync.RWMutex
	records []R
}

func (j *InMemory[R]) Read(ctx context.Context, ver uint64) (R, uint64, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(ver)
	size := len(j.records)

	switch {
	case index < size:
		return j.records[index], ver + 1, ctx.Err()
	case index > size:
		panic("offset out of range")
	default:
		var zero R
		return zero, ver, ctx.Err()
	}
}

func (j *InMemory[R]) Write(ctx context.Context, ver uint64, rec R) error {
	j.m.Lock()
	defer j.m.Unlock()

	index := int(ver)
	size := len(j.records)

	switch {
	case index < size:
		return ErrConflict
	case index > size:
		panic("offset out of range")
	default:
		j.records = append(j.records, rec)
		return ctx.Err()
	}
}
