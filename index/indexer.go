package index

import (
	"bytes"
	"context"

	"github.com/dogmatiq/veracity/journal"
)

// journalRecordIDKey is the key used to store the ID of the most recent
// fully-indexed journal record.
var journalRecordIDKey = formatKey("journal/last-record-id")

// Builder builds an index from a journal.
type Builder struct {
	Journal journal.Journal
	Index   Index
}

// Build updates the index to reflect the records from the journal.
func (b *Builder) Build(ctx context.Context) (lastRecordID []byte, err error) {
	// Find the current end of the journal so we know when we're done.
	lastID, err := b.Journal.LastID(ctx)
	if err != nil {
		return nil, err
	}

	// There are no records in the journal.
	if len(lastID) == 0 {
		return lastID, nil
	}

	// Find the last record ID that we know has been indexed fully.
	lastIndexedRecordID, err := b.Index.Get(ctx, journalRecordIDKey)
	if err != nil {
		return nil, err
	}

	// Bail if the key/value store is already up to date with the journal.
	if bytes.Equal(lastIndexedRecordID, lastID) {
		return lastID, nil
	}

	// Otherwise open the journal immediately after the last indexed record and
	// begin indexing.
	r, err := b.Journal.Open(ctx, lastIndexedRecordID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Read, unmarshal and index records until we reach the end of the journal.
	for {
		id, rec, err := journal.Read(ctx, r)
		if err != nil {
			return nil, err
		}

		p := processor{
			Index: b.Index,
		}

		if err := rec.Process(ctx, p); err != nil {
			return nil, err
		}

		if err := b.Index.Set(ctx, journalRecordIDKey, id); err != nil {
			return nil, err
		}

		if bytes.Equal(id, lastID) {
			return lastID, nil
		}
	}
}

type processor struct {
	Index Index
}

func (p processor) ProcessExecutorExecuteCommandRecord(
	ctx context.Context,
	rec *journal.ExecutorExecuteCommand,
) error {
	return addMessageToQueue(ctx, p.Index, rec.Envelope)
}

func (p processor) ProcessAggregateHandleCommandRecord(
	ctx context.Context,
	rec *journal.AggregateHandleCommand,
) error {
	return removeMessageFromQueue(ctx, p.Index, rec.MessageId)
}

func (p processor) ProcessIntegrationHandleCommandRecord(
	ctx context.Context,
	rec *journal.IntegrationHandleCommand,
) error {
	return removeMessageFromQueue(ctx, p.Index, rec.MessageId)
}

func (p processor) ProcessProcessHandleEventRecord(
	ctx context.Context,
	rec *journal.ProcessHandleEvent,
) error {
	return nil
}

func (p processor) ProcessProcessHandleTimeoutRecord(
	ctx context.Context,
	rec *journal.ProcessHandleTimeout,
) error {
	return nil
}
