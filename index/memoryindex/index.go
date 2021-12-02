package memoryindex

import (
	"context"
	"sync"
)

// Index is an in-memory implementation of the index.Index interface.
type Index struct {
	m      sync.RWMutex
	values map[string][]byte
}

// Set sets the value associated with k to v.
//
// There is no distinction between empty and non-existent values. Setting a
// key to an empty value effectively removes that key.
func (i *Index) Set(_ context.Context, k, v []byte) error {
	i.m.Lock()
	defer i.m.Unlock()

	sk := string(k)

	if len(v) == 0 {
		delete(i.values, sk)
		return nil
	}

	if i.values == nil {
		i.values = map[string][]byte{}
	}

	i.values[sk] = v
	return nil
}

// Get returns the value associated with k.
//
// It returns an empty value if k is not set.
func (i *Index) Get(_ context.Context, k []byte) (v []byte, err error) {
	i.m.RLock()
	defer i.m.RUnlock()

	return i.values[string(k)], nil
}
