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
	events []parcel.Parcel,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.events = append(s.events, events...)

	return nil
}
