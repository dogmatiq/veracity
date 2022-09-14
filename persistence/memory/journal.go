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

	return &binaryJournal[R]{
		state: state.(*journalState[R]),
	}, ctx.Err()
}

// NewJournal returns a new standalone journal.
func NewJournal[R any]() journal.Journal[R] {
	return &binaryJournal[R]{
		state: &journalState[R]{},
	}
}

// journalState stores the underlying state of a journal.
type journalState[R any] struct {
	sync.RWMutex
	Records []R
}

// binaryJournal is an implementation of journal.Journal[R] that accesses
// journal state.
type binaryJournal[R any] struct {
	state *journalState[R]
}

func (h *binaryJournal[R]) Read(ctx context.Context, ver uint64) (R, bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.RLock()
	defer h.state.RUnlock()

	index := int(ver)
	size := len(h.state.Records)

	if index < size {
		return h.state.Records[index], true, ctx.Err()
	}

	var zero R
	return zero, false, ctx.Err()
}

func (h *binaryJournal[R]) Write(ctx context.Context, v uint64, r R) (bool, error) {
	if h.state == nil {
		panic("journal is closed")
	}

	h.state.Lock()
	defer h.state.Unlock()

	index := int(v)
	size := len(h.state.Records)

	switch {
	case index < size:
		return false, ctx.Err()
	case index == size:
		h.state.Records = append(h.state.Records, r)
		return true, ctx.Err()
	default:
		panic("version out of range, this behavior would be undefined in a real journal implementation")
	}
}

func (h *binaryJournal[R]) Close() error {
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
