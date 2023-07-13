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
	onOpen   func(name string) error
}

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	if s.onOpen != nil {
		if err := s.onOpen(name); err != nil {
			return nil, err
		}
	}

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

	Begin, End journal.Position
	Records    [][]byte

	BeforeAppend func([]byte) error
	AfterAppend  func([]byte) error
}

// journalHandle is an implementation of [journal.Journal] that accesses
// in-memory journal state.
type journalHandle struct {
	state *journalState
}

func (h *journalHandle) Bounds(ctx context.Context) (begin, end journal.Position, err error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	return h.state.Begin, h.state.End, ctx.Err()
}

func (h *journalHandle) Get(ctx context.Context, pos journal.Position) ([]byte, bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	if pos < h.state.Begin || pos >= h.state.End {
		return nil, false, nil
	}

	return slices.Clone(h.state.Records[pos-h.state.Begin]), true, ctx.Err()
}

func (h *journalHandle) Range(
	ctx context.Context,
	begin journal.Position,
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
		v := start + journal.Position(i)
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
	pos := h.state.Begin
	records := h.state.Records
	h.state.RUnlock()

	for _, rec := range records {
		ok, err := fn(ctx, pos, slices.Clone(rec))
		if !ok || err != nil {
			return err
		}
		pos++
	}

	return ctx.Err()
}

func (h *journalHandle) Append(ctx context.Context, end journal.Position, rec []byte) error {
	if h.state == nil {
		panic("journal is closed")
	}

	rec = slices.Clone(rec)

	h.state.Lock()
	defer h.state.Unlock()

	if h.state.BeforeAppend != nil {
		if err := h.state.BeforeAppend(rec); err != nil {
			return err
		}
	}

	switch {
	case end < h.state.End:
		return journal.ErrConflict
	case end == h.state.End:
		h.state.Records = append(h.state.Records, rec)
		h.state.End++
	default:
		panic("position out of range, this causes undefined behavior in a 'real' journal implementation")
	}

	if h.state.AfterAppend != nil {
		if err := h.state.AfterAppend(rec); err != nil {
			return err
		}
	}

	return ctx.Err()
}

func (h *journalHandle) Truncate(ctx context.Context, end journal.Position) error {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	if end > h.state.End {
		panic("position out of range, this causes undefined behavior in a real journal implementation")
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
