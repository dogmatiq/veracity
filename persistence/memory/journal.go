package memory

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/dogmatiq/veracity/journal"
)

// JournalOpener is an implementation of journal.Opener[R] that opens in-memory
// journals.
type JournalOpener[R any] struct {
	journals sync.Map // map[string]*journalState[R]
}

// Open returns the journal at the given path.
//
// The path uniquely identifies the journal. It must not be empty. Each element
// must be a non-empty UTF-8 string consisting solely of printable Unicode
// characters, excluding whitespace. A printable character is any character from
// the Letter, Mark, Number, Punctuation or Symbol categories.
func (o *JournalOpener[R]) Open(ctx context.Context, path ...string) (journal.Journal[R], error) {
	key := keyFromJournalPath(path)
	state, ok := o.journals.Load(key)

	if !ok {
		state, _ = o.journals.LoadOrStore(
			key,
			&journalState[R]{},
		)
	}

	return &journalHandle[R]{
		state: state.(*journalState[R]),
	}, ctx.Err()
}

// NewJournal returns a new standalone journal.
func NewJournal[R any]() journal.Journal[R] {
	return &journalHandle[R]{
		state: &journalState[R]{},
	}
}

// journalState stores the underlying state of a journal.
type journalState[R any] struct {
	sync.RWMutex

	Begin   uint64
	End     uint64
	Records []R
}

// journalHandle is an implementation of journal.Journal[R] that accesses
// journal state.
type journalHandle[R any] struct {
	state *journalState[R]
}

func (h *journalHandle[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	if ver < h.state.Begin || ver >= h.state.End {
		var zero R
		return zero, false, nil
	}

	return h.state.Records[ver-h.state.Begin], true, ctx.Err()
}

func (h *journalHandle[R]) ReadOldest(ctx context.Context) (uint64, R, bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	if h.state.Begin == h.state.End {
		var zero R
		return 0, zero, false, ctx.Err()
	}

	return h.state.Begin, h.state.Records[0], true, ctx.Err()
}

func (h *journalHandle[R]) Write(ctx context.Context, ver uint64, rec R) (bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	switch {
	case ver < h.state.End:
		return false, ctx.Err()
	case ver == h.state.End:
		h.state.Records = append(h.state.Records, rec)
		h.state.End++
		return true, ctx.Err()
	default:
		panic("version out of range, this behavior would be undefined in a real journal implementation")
	}
}

func (h *journalHandle[R]) Truncate(ctx context.Context, ver uint64) error {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	if ver > h.state.End {
		panic("version out of range, this behavior would be undefined in a real journal implementation")
	}

	if ver > h.state.Begin {
		h.state.Records = h.state.Records[ver-h.state.Begin:]
		h.state.Begin = ver
	}

	return ctx.Err()
}

func (h *journalHandle[R]) Close() error {
	if h.state == nil {
		return errors.New("journal is already closed")
	}

	h.state = nil

	return nil
}

func keyFromJournalPath(path []string) string {
	if len(path) == 0 {
		panic("path must not be empty")
	}

	var w strings.Builder

	for _, elem := range path {
		if len(elem) == 0 {
			panic("path element must not be empty")
		}

		if w.Len() > 0 {
			w.WriteByte('/')
		}

		for _, r := range elem {
			if r == '/' || r == '\\' {
				w.WriteByte('\\')
			}

			w.WriteRune(r)
		}
	}

	return w.String()
}
