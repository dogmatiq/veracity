package journal

import (
	"bytes"
	"context"

	"github.com/dogmatiq/veracity/journal/internal/indexpb"
	"github.com/dogmatiq/veracity/persistence"
	"google.golang.org/protobuf/proto"
)

// Committer commits journal entries to the index as they are written to the
// journal.
type Committer struct {
	Journal persistence.Journal
	Index   persistence.KeyValueStore

	commit indexpb.Commit
	synced bool
}

var (
	// commitKey is the key used to store information about the most-recent
	// fully-committed journal record.
	commitKey = makeKey("commit")
)

// SyncIndex synchronizes the index with the journal.
//
// The index must be synchronized before new records are appended.
//
// It returns the ID of the last record in the journal.
func (c *Committer) SyncIndex(ctx context.Context) ([]byte, error) {
	if err := c.getProto(ctx, commitKey, &c.commit); err != nil {
		return nil, err
	}

	r, err := c.Journal.Open(ctx, c.commit.RecordId)
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
			return c.commit.RecordId, nil
		}

		if err := proto.Unmarshal(data, &container); err != nil {
			return nil, err
		}

		rec := container.Unpack()
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
// It panics if the index has not been synchronized.
func (c *Committer) Append(
	ctx context.Context,
	prevID []byte,
	rec Record,
) ([]byte, error) {
	if !c.synced {
		panic("Apply() called without first calling SyncIndex()")
	}

	data, err := proto.Marshal(rec.Pack())
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

	return id, nil
}

// apply updates the index to reflrect the next journal record.
func (c *Committer) apply(ctx context.Context, id []byte, rec Record) error {
	if err := rec.AcceptVisitor(
		ctx,
		id,
		indexer{c},
	); err != nil {
		return err
	}

	c.commit.RecordId = id

	return c.setProto(ctx, commitKey, &c.commit)
}

// indexer is an implementation of journal.RecordVisitor that applies records to
// the index.
type indexer struct {
	committer *Committer
}

func (i indexer) VisitExecutorExecuteCommandRecord(ctx context.Context, id []byte, rec *ExecutorExecuteCommand) error {
	return i.committer.addMessageToQueue(ctx, rec.Envelope)
}

func (i indexer) VisitAggregateHandleCommandRecord(
	ctx context.Context,
	id []byte,
	rec *AggregateHandleCommand,
) error {
	if err := i.committer.removeMessageFromQueue(ctx, rec.MessageId); err != nil {
		return err
	}

	return nil
}

func (i indexer) VisitIntegrationHandleCommandRecord(
	ctx context.Context,
	id []byte,
	rec *IntegrationHandleCommand,
) error {
	if err := i.committer.removeMessageFromQueue(ctx, rec.MessageId); err != nil {
		return err
	}

	return nil
}

func (i indexer) VisitProcessHandleEventRecord(
	ctx context.Context,
	id []byte,
	rec *ProcessHandleEvent,
) error {
	return nil
}

func (i indexer) VisitProcessHandleTimeoutRecord(
	ctx context.Context,
	id []byte,
	rec *ProcessHandleTimeout,
) error {
	return nil
}

// setProto sets the value of k to v's binary representation.
func (c *Committer) setProto(ctx context.Context, k []byte, v proto.Message) error {
	data, err := proto.Marshal(v)
	if err != nil {
		return err
	}

	return c.Index.Set(ctx, k, data)
}

// getProto reads the value of k into v.
func (c *Committer) getProto(ctx context.Context, k []byte, v proto.Message) error {
	data, err := c.Index.Get(ctx, k)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, v)
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
