package journal

import (
	"bytes"
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

// makeKey returns a key made from slash-separated parts.
func makeKey(parts ...string) []byte {
	var buf bytes.Buffer

	for i, p := range parts {
		if i > 0 {
			buf.WriteByte('/')
		}

		buf.WriteString(p)
	}

	return buf.Bytes()
}

var (
	// metaDataKey is the key used to store information about the most-recent
	// fully-committed journal record.
	metaDataKey = makeKey("meta-data")
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
// It may only be called after Sync() has succeeded. If an error is returned the
// index must be re-sychronized.
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
		c.synced = false
		return nil, err
	}

	if err := c.apply(ctx, id, rec); err != nil {
		c.synced = false
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
	case *ExecutorExecuteCommand:
		err = c.applyExecutorExecuteCommand(ctx, rec)
	case *AggregateHandleCommand:
		err = c.applyAggregateHandleCommand(ctx, rec)
	case *IntegrationHandleCommand:
		err = c.applyIntegrationHandleCommand(ctx, rec)
	case *ProcessHandleEvent:
		err = c.applyProcessHandleEvent(ctx, rec)
	case *ProcessHandleTimeout:
		err = c.applyProcessHandleTimeout(ctx, rec)
	}

	if err != nil {
		return err
	}

	c.metaData.CommittedRecordId = id

	return c.set(ctx, metaDataKey, &c.metaData)
}

func (c *Committer) applyExecutorExecuteCommand(ctx context.Context, rec *ExecutorExecuteCommand) error {
	return c.addMessageToQueue(ctx, rec.Envelope)
}

func (c *Committer) applyAggregateHandleCommand(ctx context.Context, rec *AggregateHandleCommand) error {
	if err := c.removeMessageFromQueue(ctx, rec.MessageId); err != nil {
		return err
	}

	return nil
}

func (c *Committer) applyIntegrationHandleCommand(ctx context.Context, rec *IntegrationHandleCommand) error {
	if err := c.removeMessageFromQueue(ctx, rec.MessageId); err != nil {
		return err
	}

	return nil
}

func (c *Committer) applyProcessHandleEvent(ctx context.Context, rec *ProcessHandleEvent) error {
	return nil
}

func (c *Committer) applyProcessHandleTimeout(ctx context.Context, rec *ProcessHandleTimeout) error {
	return nil
}
