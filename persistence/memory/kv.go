package memory

import (
	"context"
	"sync"
)

// KeyValueStore is an in-memory associative database that creates mappings
// between UTF-8 keys and opaque binary values.
//
// It is an implementation of the persistence.KeyValueStore interface intended
// for testing purposes.
//
// Keys are structured as Unix-like paths, including a leading slash. Each path
// component is a URL encoded UTF-8 string.
//
// The store MAY parse keys to structure data heirarchically, however two keys
// must only considered equivalent if they are equal byte-for-byte.
type KeyValueStore struct {
	m      sync.RWMutex
	values map[string][]byte
}

// Set associates the value v with the key k.
//
// Any value already associated with k is replaced.
//
// Setting v to an empty (or nil) slice deletes any existing association.
func (s *KeyValueStore) Set(_ context.Context, k string, v []byte) error {
	s.m.Lock()
	defer s.m.Unlock()

	if len(v) == 0 {
		delete(s.values, k)
		return nil
	}

	if s.values == nil {
		s.values = map[string][]byte{}
	}

	s.values[k] = v
	return nil
}

// Get returns the value associated with k.
//
// It returns an empty value if k is not set.
func (s *KeyValueStore) Get(_ context.Context, k string) (v []byte, err error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.values[k], nil
}
