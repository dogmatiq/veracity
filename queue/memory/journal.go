package memory

import (
	"context"
	"sync"

	"github.com/dogmatiq/veracity/queue"
)

// Journal is an in-memory implementation of the Journal interface.
type Journal struct {
	m       sync.RWMutex
	entries []queue.JournalEntry
}

func (j *Journal) Read(
	ctx context.Context,
	offset uint64,
) ([]queue.JournalEntry, uint64, error) {
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

func (j *Journal) Write(
	ctx context.Context,
	offset uint64,
	r queue.JournalEntry,
) error {
	j.m.Lock()
	defer j.m.Unlock()

	count := uint64(len(j.entries))

	if offset == count {
		j.entries = append(j.entries, r)
		return nil
	}

	if offset < count {
		return queue.ErrConflict
	}

	panic("offset out of range")
}
