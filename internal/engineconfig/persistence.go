package engineconfig

import (
	"net/url"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/veracity/internal/telemetry/instrumentedpersistence"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// journalStoreDSN is the DSN describing which journal store to use.
var journalStoreDSN = ferrite.
	URL("VERACITY_JOURNAL_DSN", "the DSN of the journal store").
	Optional(ferrite.WithRegistry(FerriteRegistry))

	// keyValueStoreDSN is the DSN describing which key/value store to use.
var keyValueStoreDSN = ferrite.
	URL("VERACITY_KV_DSN", "the DSN of the key/value store").
	Optional(ferrite.WithRegistry(FerriteRegistry))

// journalStoreFromDSN returns the journal store described by the given DSN.
func journalStoreFromDSN(dsn *url.URL) journal.Store {
	panic("not implemented")
}

// keyValueStoreFromDSN returns the key/value store described by the given DSN.
func keyValueStoreFromDSN(dsn *url.URL) kv.Store {
	panic("not implemented")
}

func (c *Config) finalizePersistence() {
	if c.UseEnv {
		if c.Persistence.Journals == nil {
			if dsn, ok := journalStoreDSN.Value(); ok {
				c.Persistence.Journals = journalStoreFromDSN(dsn)
			}
		}

		if c.Persistence.Keyspaces == nil {
			if dsn, ok := keyValueStoreDSN.Value(); ok {
				c.Persistence.Keyspaces = keyValueStoreFromDSN(dsn)
			}
		}
	}

	if c.Persistence.Journals == nil {
		panic("no journal store is configured, set VERACITY_JOURNAL_DSN or provide the WithJournalStore() option")
	}

	if c.Persistence.Keyspaces == nil {
		panic("no key/value store is configured, set VERACITY_KV_DSN or provide the WithKeyValueStore() option")
	}

	c.Persistence.Journals = &instrumentedpersistence.JournalStore{
		Next:      c.Persistence.Journals,
		Telemetry: c.Telemetry,
	}

	c.Persistence.Keyspaces = &instrumentedpersistence.KeyValueStore{
		Next:      c.Persistence.Keyspaces,
		Telemetry: c.Telemetry,
	}
}
