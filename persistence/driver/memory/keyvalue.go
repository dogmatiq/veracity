package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// KeyValueStore is an implementation of kv.Store that stores keyspaces in
// memory.
type KeyValueStore struct {
	keyspaces sync.Map // map[string]*keyspaceState
}

// Open returns the keyspace at the given path.
//
// The path uniquely identifies the keyspace. It must not be empty. Each
// element must be a non-empty UTF-8 string consisting solely of printable
// Unicode characters, excluding whitespace. A printable character is any
// character from the Letter, Mark, Number, Punctuation or Symbol
// categories.
func (s *KeyValueStore) Open(ctx context.Context, path ...string) (kv.Keyspace, error) {
	key := pathkey.New(path)
	state, ok := s.keyspaces.Load(key)

	if !ok {
		state, _ = s.keyspaces.LoadOrStore(
			key,
			&keyspaceState{},
		)
	}

	return &keyspaceHandle{
		state: state.(*keyspaceState),
	}, ctx.Err()
}

type keyspaceState struct {
	Map sync.Map // map[string][]byte
}

type keyspaceHandle struct {
	state *keyspaceState
}

func (h *keyspaceHandle) Get(ctx context.Context, k []byte) (v []byte, err error) {
	if h.state == nil {
		panic("keyspace is closed")
	}

	if x, ok := h.state.Map.Load(string(k)); ok {
		return x.([]byte), ctx.Err()
	}

	return nil, ctx.Err()
}

func (h *keyspaceHandle) Set(ctx context.Context, k, v []byte) error {
	if h.state == nil {
		panic("keyspace is closed")
	}

	if len(v) == 0 {
		h.state.Map.Delete(string(k))
	} else {
		h.state.Map.Store(string(k), v)
	}

	return ctx.Err()
}

func (h *keyspaceHandle) Close() error {
	if h.state == nil {
		return errors.New("keyspace is already closed")
	}

	h.state = nil

	return nil
}
