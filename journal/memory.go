package journal

import (
	"context"
	"errors"
	"sync"
)

// InMemoryOpener is an implementation of JournalOpener that opens in-memory
// journals.
type InMemoryOpener[R any] struct {
	m    sync.Mutex
	data map[string]*sharedData[R]
}

// OpenJournal opens the journal identified by the given key.
func (o *InMemoryOpener[R]) OpenJournal(
	ctx context.Context,
	id string,
) (Journal[R], error) {
	o.m.Lock()
	defer o.m.Unlock()

	data, ok := o.data[id]

	if !ok {
		if o.data == nil {
			o.data = map[string]*sharedData[R]{}
		}

		data = &sharedData[R]{}
		o.data[id] = data
	}

	return &InMemory[R]{
		data: data,
	}, nil
}

// sharedData is an in-memory collection of journal sharedData that is
// sharedData between instances of in-memory journals that refer to the same
// journal key.
type sharedData[R any] struct {
	m       sync.RWMutex
	records []R
}

// InMemory is an in-memory journal.
type InMemory[R any] struct {
	m      sync.Mutex
	data   *sharedData[R]
	closed bool
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *InMemory[R]) Read(ctx context.Context, v uint64) (R, bool, error) {
	data := j.get()

	data.m.RLock()
	defer data.m.RUnlock()

	index := int(v)
	size := len(data.records)

	if index < size {
		return data.records[index], true, ctx.Err()
	}

	var zero R
	return zero, false, ctx.Err()
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// It panics if v > current.
func (j *InMemory[R]) Write(ctx context.Context, v uint64, r R) (bool, error) {
	data := j.get()

	data.m.Lock()
	defer data.m.Unlock()

	index := int(v)
	size := len(data.records)

	switch {
	case index < size:
		return false, ctx.Err()
	case index == size:
		data.records = append(data.records, r)
		return true, ctx.Err()
	default:
		panic("version out of range, this behavior would be undefined in a real journal implementation")
	}
}

// Close closes the journal.
func (j *InMemory[R]) Close() error {
	j.m.Lock()
	defer j.m.Unlock()

	if j.closed {
		return errors.New("journal is already closed")
	}

	j.closed = true
	j.data = nil

	return nil
}

func (j *InMemory[R]) get() *sharedData[R] {
	j.m.Lock()
	defer j.m.Unlock()

	if j.closed {
		panic("journal is closed")
	}

	if j.data == nil {
		// This journal was not obtained from an InMemoryOpener, so we just
		// lazily construct some record storage to allow it to work
		// independently.
		j.data = &sharedData[R]{}
	}

	return j.data
}
