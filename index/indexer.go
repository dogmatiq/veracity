package index

import (
	"context"

	"github.com/dogmatiq/veracity/journal"
)

// Builder builds an index from a journal.
type Builder struct {
	Journal journal.Journal
	Index   Index
}

// Build updates the index to reflect the records from the journal.
func (b *Builder) Build(ctx context.Context) (lastRecordID []byte, err error) {
	j, err := loadJournal(ctx, b.Index)
	if err != nil {
		return nil, err
	}

	// Read from the journal immediately after the last indexed record and begin
	// indexing.
	j.CommittedRecordId, err = journal.VisitRecords(
		ctx,
		b.Journal,
		j.CommittedRecordId,
		indexer{b.Index},
	)
	if err != nil {
		return nil, err
	}

	if err := saveJournal(ctx, b.Index, j); err != nil {
		return nil, err
	}

	return j.CommittedRecordId, nil
}

type indexer struct {
	Index Index
}

func (i indexer) VisitExecutorExecuteCommandRecord(
	ctx context.Context,
	id []byte,
	rec *journal.ExecutorExecuteCommand,
) error {
	return addMessageToQueue(ctx, i.Index, rec.Envelope)
}

func (i indexer) VisitAggregateHandleCommandRecord(
	ctx context.Context,
	id []byte,
	rec *journal.AggregateHandleCommand,
) error {
	return removeMessageFromQueue(ctx, i.Index, rec.MessageId)
}

func (i indexer) VisitIntegrationHandleCommandRecord(
	ctx context.Context,
	id []byte,
	rec *journal.IntegrationHandleCommand,
) error {
	return removeMessageFromQueue(ctx, i.Index, rec.MessageId)
}

func (i indexer) VisitProcessHandleEventRecord(
	ctx context.Context,
	id []byte,
	rec *journal.ProcessHandleEvent,
) error {
	return nil
}

func (i indexer) VisitProcessHandleTimeoutRecord(
	ctx context.Context,
	id []byte,
	rec *journal.ProcessHandleTimeout,
) error {
	return nil
}
