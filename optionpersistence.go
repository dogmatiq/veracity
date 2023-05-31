package veracity

import (
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// WithJournalStore is an [EngineOption] that sets the journal store used by the
// engine.
func WithJournalStore(s journal.Store) EngineOption {
	return option{
		engineOption: func(e *Engine) {
			e.journals = s
		},
	}
}

// WithKeyValueStore is an [EngineOption] that sets the key/value store used by
// the engine.
func WithKeyValueStore(s kv.Store) EngineOption {
	return option{
		engineOption: func(e *Engine) {
			e.keyspaces = s
		},
	}
}
