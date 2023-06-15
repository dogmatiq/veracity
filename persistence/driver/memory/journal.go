package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dogmatiq/veracity/persistence/journal"
	"golang.org/x/exp/slices"
)

// JournalStore is an implementation of [journal.Store] that stores journals in
// memory.
type JournalStore struct {
	journals sync.Map // map[string]*journalState
}

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	state, ok := s.journals.Load(name)

	if !ok {
		state, _ = s.journals.LoadOrStore(
			name,
			&journalState{},
		)
	}

	return &journalHandle{
		state: state.(*journalState),
	}, ctx.Err()
}

// NewJournal returns a new standalone journal.
func NewJournal() journal.Journal {
	return &journalHandle{
		state: &journalState{},
	}
}

// journalState stores the underlying state of a journal.
type journalState struct {
	sync.RWMutex

	Begin   uint64
	End     uint64
	Records [][]byte

	BeforeAppend func([]byte) error
	AfterAppend  func([]byte) error
}

// journalHandle is an implementation of [journal.Journal] that accesses
// in-memory journal state.
type journalHandle struct {
	state *journalState
}

func (h *journalHandle) Get(ctx context.Context, offset uint64) ([]byte, bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	if offset < h.state.Begin || offset >= h.state.End {
		return nil, false, nil
	}

	return slices.Clone(h.state.Records[offset-h.state.Begin]), true, ctx.Err()
}

func (h *journalHandle) Range(
	ctx context.Context,
	begin uint64,
	fn journal.RangeFunc,
) error {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	first := h.state.Begin
	records := h.state.Records
	h.state.RUnlock()

	if first > begin {
		return fmt.Errorf("cannot range over truncated records")
	}

	start := begin - first
	for i, rec := range records[start:] {
		v := start + uint64(i)
		ok, err := fn(ctx, v, slices.Clone(rec))
		if !ok || err != nil {
			return err
		}
		begin++
	}

	return ctx.Err()
}

func (h *journalHandle) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	offset := h.state.Begin
	records := h.state.Records
	h.state.RUnlock()

	for _, rec := range records {
		ok, err := fn(ctx, offset, slices.Clone(rec))
		if !ok || err != nil {
			return err
		}
		offset++
	}

	return ctx.Err()
}

func (h *journalHandle) Append(ctx context.Context, offset uint64, rec []byte) (bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	rec = slices.Clone(rec)

	h.state.Lock()
	defer h.state.Unlock()

	if h.state.BeforeAppend != nil {
		if err := h.state.BeforeAppend(rec); err != nil {
			return false, err
		}
	}

	switch {
	case offset < h.state.End:
		return false, ctx.Err()
	case offset == h.state.End:
		h.state.Records = append(h.state.Records, rec)
		h.state.End++
	default:
		panic("offset out of range, this behavior may be undefined in a 'real' journal implementation")
	}

	if h.state.AfterAppend != nil {
		if err := h.state.AfterAppend(rec); err != nil {
			return false, err
		}
	}

	return true, ctx.Err()
}

func (h *journalHandle) Truncate(ctx context.Context, end uint64) error {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	if end > h.state.End {
		panic("offset out of range, this behavior may be undefined in a real journal implementation")
	}

	if end > h.state.Begin {
		h.state.Records = h.state.Records[end-h.state.Begin:]
		h.state.Begin = end
	}

	return ctx.Err()
}

func (h *journalHandle) Close() error {
	if h.state == nil {
		return errors.New("journal is already closed")
	}

	h.state = nil

	return nil
}
