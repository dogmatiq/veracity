package journal

import (
	"context"

	"github.com/dogmatiq/veracity/journal/internal/indexpb"
	"github.com/dogmatiq/veracity/persistence"
)

// Committer commits journal entries to the index as they are written to the
// journal.
type Committer struct {
	Journal     persistence.Journal
	Index       persistence.KeyValueStore
	Marshaler   Marshaler
	Unmarshaler Unmarshaler

	metaData indexpb.MetaData
	synced   bool
}

var (
	// metaDataKey is the key used to store information about the most-recent
	// fully-committed journal record.
	metaDataKey = persistence.Key("meta-data")
)

// Sync synchronizes the index with the journal.
//
// The index must be synchronized before new records are appended.
//
// It returns the ID of the last record in the journal.
func (c *Committer) Sync(ctx context.Context) ([]byte, error) {
	if err := c.get(ctx, metaDataKey, &c.metaData); err != nil {
		return nil, err
	}

	r, err := c.Journal.Open(ctx, c.metaData.CommittedRecordId)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var container RecordContainer

	for {
		id, data, ok, err := r.Next(ctx)
		if err != nil {
			return nil, err
		}

		if !ok {
			c.synced = true
			return c.metaData.CommittedRecordId, nil
		}

		if err := c.unmarshal(data, &container); err != nil {
			return nil, err
		}

		rec := container.unpack()
		if err := c.apply(ctx, id, rec); err != nil {
			return nil, err
		}
	}
}

// Append appends a record to the journal and updates the index.
//
// prevID must be the ID of the most recent record, or an empty slice if the
// journal is currently empty; otherwise, the append operation fails.
//
// It returns the ID of the newly appended record.
//
// The index must be synchronized with the journal by a prior successful call to
// Sync(). If Append() returns a non-nil error the index must be re-synchronized
// before calling Append() again.
func (c *Committer) Append(
	ctx context.Context,
	prevID []byte,
	rec Record,
) ([]byte, error) {
	if !c.synced {
		panic("Apply() called without first calling Sync()")
	}

	// Pre-emptively mark the index as out-of-sync.
	c.synced = false

	data, err := c.marshal(rec.pack())
	if err != nil {
		return nil, err
	}

	id, err := c.Journal.Append(ctx, prevID, data)
	if err != nil {
		return nil, err
	}

	if err := c.apply(ctx, id, rec); err != nil {
		return nil, err
	}

	// All writes succeeded, mark the index as in-sync.
	c.synced = true

	return id, nil
}

// apply updates the index to reflrect the next journal record.
func (c *Committer) apply(ctx context.Context, id []byte, rec Record) error {
	var err error

	switch rec := rec.(type) {
	case *CommandEnqueued:
		err = c.commandEnqueued(ctx, id, rec)
	case *CommandHandledByAggregate:
		err = c.commandHandledByAggregate(ctx, id, rec)
	case *CommandHandledByIntegration:
		err = c.commandHandledByIntegration(ctx, id, rec)
	case *EventHandledByProcess:
		err = c.eventHandledByProcess(ctx, id, rec)
	case *TimeoutHandledByProcess:
		err = c.timeoutHandledByProcess(ctx, id, rec)
	default:
		panic("unrecognized record type")
	}

	if err != nil {
		return err
	}

	c.metaData.CommittedRecordId = id

	return c.set(ctx, metaDataKey, &c.metaData)
}
