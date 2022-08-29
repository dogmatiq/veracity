package queue

import (
	"context"
	"sync"
)

type MemoryJournal struct {
	m       sync.RWMutex
	entries []JournalEntry
}

func (j *MemoryJournal) Read(
	ctx context.Context,
	offset uint64,
) ([]JournalEntry, uint64, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(offset)
	count := len(j.entries)

	if index > count {
		panic("offset out of range")
	}

	if index == count {
		return nil, offset + 1, nil
	}

	return j.entries[index : index+1], offset + 1, nil
}

func (j *MemoryJournal) Write(
	ctx context.Context,
	offset uint64,
	r JournalEntry,
) error {
	j.m.Lock()
	defer j.m.Unlock()

	count := uint64(len(j.entries))

	if offset == count {
		j.entries = append(j.entries, r)
		return nil
	}

	if offset < count {
		return ErrConflict
	}

	panic("offset out of range")
}
