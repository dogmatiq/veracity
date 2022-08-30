package journal

import (
	"context"
	"sync"
)

// InMemory is an in-memory append-only log.
type InMemory[E any] struct {
	m       sync.RWMutex
	entries []E
}

func (j *InMemory[E]) Read(
	ctx context.Context,
	offset uint64,
) ([]E, uint64, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(offset)
	size := len(j.entries)

	switch {
	case index < size:
		return j.entries[index : index+1], offset + 1, ctx.Err()
	case index > size:
		panic("offset out of range")
	default:
		return nil, offset, ctx.Err()
	}
}

func (j *InMemory[E]) Write(
	ctx context.Context,
	offset uint64,
	entry E,
) error {
	j.m.Lock()
	defer j.m.Unlock()

	index := int(offset)
	size := len(j.entries)

	switch {
	case index < size:
		return ErrConflict
	case index > size:
		panic("offset out of range")
	default:
		j.entries = append(j.entries, entry)
		return ctx.Err()
	}
}
