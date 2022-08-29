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
	offset *uint64,
) ([]E, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(*offset)
	count := len(j.entries)

	if index > count {
		panic("offset out of range")
	}

	if index == count {
		return nil, nil
	}

	*offset++

	return j.entries[index : index+1], nil
}

func (j *InMemory[E]) Write(
	ctx context.Context,
	offset uint64,
	entry E,
) error {
	j.m.Lock()
	defer j.m.Unlock()

	count := uint64(len(j.entries))

	if offset == count {
		j.entries = append(j.entries, entry)
		return nil
	}

	if offset < count {
		return ErrConflict
	}

	panic("offset out of range")
}
