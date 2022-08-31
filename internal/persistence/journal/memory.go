package journal

import (
	"context"
	"sync"
)

// InMemory is an in-memory journal.
type InMemory[R any] struct {
	m       sync.RWMutex
	records []R
}

func (j *InMemory[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(ver)
	size := len(j.records)

	if index < size {
		return j.records[index], true, ctx.Err()
	}

	var zero R
	return zero, false, ctx.Err()
}

func (j *InMemory[R]) Write(ctx context.Context, ver uint64, rec R) (bool, error) {
	j.m.Lock()
	defer j.m.Unlock()

	index := int(ver)
	size := len(j.records)

	switch {
	case index < size:
		return false, ctx.Err()
	case index == size:
		j.records = append(j.records, rec)
		return true, ctx.Err()
	default:
		panic("version out of range, this behavior would be undefined in a real journal implementation")
	}
}
