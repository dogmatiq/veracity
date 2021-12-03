package memory

import (
	"context"
	"sync"
)

// KeyValueStore is an in-memory associative database that creates mappings
// between opaque binary keys and values.
//
// It is an implementation of the persistence.KeyValueStore interface intended
// for testing purposes.
type KeyValueStore struct {
	m      sync.RWMutex
	values map[string][]byte
}

// Set associates the value v with the key k.
//
// Any value already associated with k is replaced.
//
// Setting v to an empty (or nil) slice deletes any existing association.
func (s *KeyValueStore) Set(_ context.Context, k, v []byte) error {
	s.m.Lock()
	defer s.m.Unlock()

	sk := string(k)

	if len(v) == 0 {
		delete(s.values, sk)
		return nil
	}

	if s.values == nil {
		s.values = map[string][]byte{}
	}

	s.values[sk] = v
	return nil
}

// Get returns the value associated with k.
//
// It returns an empty value if k is not set.
func (s *KeyValueStore) Get(_ context.Context, k []byte) (v []byte, err error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.values[string(k)], nil
}
