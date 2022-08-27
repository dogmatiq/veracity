package aggregate

import (
	"context"
	"errors"
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

type MemoryJournal struct {
	m       sync.RWMutex
	records []Record
}

func (j *MemoryJournal) Read(
	ctx context.Context,
	hk, id string,
	offset uint64,
) ([]Record, error) {
	j.m.RLock()
	defer j.m.RUnlock()

	index := int(offset)
	count := len(j.records)

	if index > count {
		panic("offset out of range")
	}

	if index == count {
		return nil, nil
	}

	return j.records[index : index+1], nil

}

func (j *MemoryJournal) Write(
	ctx context.Context,
	hk, id string,
	offset uint64,
	r Record,
) error {
	j.m.Lock()
	defer j.m.Unlock()

	count := uint64(len(j.records))

	if offset == count {
		j.records = append(j.records, r)
		return nil
	}

	if offset < count {
		return errors.New("optimistic concurrency failure")
	}

	panic("offset out of range")
}
