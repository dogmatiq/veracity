package engineconfig

import (
	"net/url"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/kv"
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
func journalStoreFromDSN(*url.URL) journal.Store {
	panic("not implemented")
}

// keyValueStoreFromDSN returns the key/value store described by the given DSN.
func keyValueStoreFromDSN(*url.URL) kv.Store {
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

	c.Persistence.Journals = journal.WithTelemetry(
		c.Persistence.Journals,
		c.Telemetry.TracerProvider,
		c.Telemetry.MeterProvider,
		c.Telemetry.Logger,
	)

	c.Persistence.Keyspaces = kv.WithTelemetry(
		c.Persistence.Keyspaces,
		c.Telemetry.TracerProvider,
		c.Telemetry.MeterProvider,
		c.Telemetry.Logger,
	)
}
