package aggregate

import (
	"context"
	"sync"

	"github.com/dogmatiq/veracity/parcel"
)

type MemoryEventStream struct {
	m      sync.RWMutex
	events []parcel.Parcel
}

func (s *MemoryEventStream) Read(
	ctx context.Context,
	offset uint64,
) ([]parcel.Parcel, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	index := int(offset)
	count := len(s.events)

	if index > count {
		panic("offset out of range")
	}

	if index == count {
		return nil, nil
	}

	return s.events[index : index+1], nil
}

func (s *MemoryEventStream) Write(
	ctx context.Context,
	ev parcel.Parcel,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.events = append(s.events, ev)

	return nil
}

func (s *MemoryEventStream) WriteAtOffset(
	ctx context.Context,
	offset uint64,
	ev parcel.Parcel,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	count := uint64(len(s.events))

	if offset == count {
		s.events = append(s.events, ev)
		return nil
	}

	if offset < count {
		return ErrConflict
	}

	panic("offset out of range")
}

type MemoryJournal struct {
	m       sync.RWMutex
	entries []JournalEntry
}

func (j *MemoryJournal) Read(
	ctx context.Context,
	hk, id string,
	offset uint64,
) ([]JournalEntry, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(offset)
	count := len(j.entries)

	if index > count {
		panic("offset out of range")
	}

	if index == count {
		return nil, nil
	}

	return j.entries[index : index+1], nil

}

func (j *MemoryJournal) Write(
	ctx context.Context,
	hk, id string,
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