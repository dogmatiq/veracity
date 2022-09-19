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
	sync.RWMutex

	Values map[string][]byte

	BeforeSet func(k, v []byte) error
	AfterSet  func(k, v []byte) error
}

type keyspaceHandle struct {
	state *keyspaceState
}

func (h *keyspaceHandle) Get(ctx context.Context, k []byte) (v []byte, err error) {
	if h.state == nil {
		panic("keyspace is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	return h.state.Values[string(k)], ctx.Err()
}

func (h *keyspaceHandle) Set(ctx context.Context, k, v []byte) error {
	if h.state == nil {
		panic("keyspace is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	if h.state.BeforeSet != nil {
		if err := h.state.BeforeSet(k, v); err != nil {
			return err
		}
	}

	if len(v) == 0 {
		delete(h.state.Values, string(k))
	} else {
		if h.state.Values == nil {
			h.state.Values = map[string][]byte{}
		}

		h.state.Values[string(k)] = v
	}

	if h.state.AfterSet != nil {
		if err := h.state.AfterSet(k, v); err != nil {
			return err
		}
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
