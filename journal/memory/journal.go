package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/dogmatiq/veracity/journal"
)

// Journal is an in-memory journal.
type Journal[R any] struct {
	m      sync.Mutex
	data   *data[R]
	closed bool
}

// sharedData is an in-memory collection of journal sharedData that is
// sharedData between instances of in-memory journals that refer to the same
// journal key.
type data[R any] struct {
	m       sync.RWMutex
	records []R
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *Journal[R]) Read(ctx context.Context, v uint64) (R, bool, error) {
	d := j.get()

	d.m.RLock()
	defer d.m.RUnlock()

	index := int(v)
	size := len(d.records)

	if index < size {
		return d.records[index], true, ctx.Err()
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
func (j *Journal[R]) Write(ctx context.Context, v uint64, r R) (bool, error) {
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
func (j *Journal[R]) Close() error {
	j.m.Lock()
	defer j.m.Unlock()

	if j.closed {
		return errors.New("journal is already closed")
	}

	j.closed = true
	j.data = nil

	return nil
}

func (j *Journal[R]) get() *data[R] {
	j.m.Lock()
	defer j.m.Unlock()

	if j.closed {
		panic("journal is closed")
	}

	if j.data == nil {
		// This journal was not obtained from a JournalOpener, so we just lazily
		// construct some record storage to allow it to work independently.
		j.data = &data[R]{}
	}

	return j.data
}

// JournalOpener is an implementation of JournalOpener that opens in-memory journals.
type JournalOpener[R any] struct {
	m    sync.Mutex
	data map[string]*data[R]
}

// Open opens the journal identified by the given key.
func (o *JournalOpener[R]) Open(ctx context.Context, key string) (journal.Journal[R], error) {
	o.m.Lock()
	defer o.m.Unlock()

	d, ok := o.data[key]

	if !ok {
		if o.data == nil {
			o.data = map[string]*data[R]{}
		}

		d = &data[R]{}
		o.data[key] = d
	}

	return &Journal[R]{data: d}, nil
}
