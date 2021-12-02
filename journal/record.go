package journal

import (
	"context"
)

// Record is an interface for a journal record.
type Record interface {
	// AcceptVisitor calls the VisitXXX() method on v that relates to this
	// record type.
	AcceptVisitor(context.Context, []byte, Visitor) error

	// Pack returns a container that contains this record.
	Pack() *RecordContainer
}

// Visitor is an interface for processing records in a journal.
type Visitor interface {
	VisitExecutorExecuteCommandRecord(context.Context, []byte, *ExecutorExecuteCommand) error
	VisitAggregateHandleCommandRecord(context.Context, []byte, *AggregateHandleCommand) error
	VisitIntegrationHandleCommandRecord(context.Context, []byte, *IntegrationHandleCommand) error
	VisitProcessHandleEventRecord(context.Context, []byte, *ProcessHandleEvent) error
	VisitProcessHandleTimeoutRecord(context.Context, []byte, *ProcessHandleTimeout) error
}

// AcceptVisitor calls v.VisitExecutorExecuteCommandRecord(ctx, id, r).
func (r *ExecutorExecuteCommand) AcceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitExecutorExecuteCommandRecord(ctx, id, r)
}

// AcceptVisitor calls v.VisitAggregateHandleCommandRecord(ctx, id, r).
func (r *AggregateHandleCommand) AcceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitAggregateHandleCommandRecord(ctx, id, r)
}

// AcceptVisitor calls v.VisitIntegrationHandleCommandRecord(ctx, id, r).
func (r *IntegrationHandleCommand) AcceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitIntegrationHandleCommandRecord(ctx, id, r)
}

// AcceptVisitor calls v.VisitProcessHandleEventRecord(ctx, id, r).
func (r *ProcessHandleEvent) AcceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitProcessHandleEventRecord(ctx, id, r)
}

// AcceptVisitor calls v.VisitProcessHandleTimeoutRecord(ctx, id, r).
func (r *ProcessHandleTimeout) AcceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitProcessHandleTimeoutRecord(ctx, id, r)
}

// Pack returns a container that contains this record.
func (r *ExecutorExecuteCommand) Pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ExecutorExecuteCommand{
			ExecutorExecuteCommand: r,
		},
	}
}

// Pack returns a container that contains this record.
func (r *AggregateHandleCommand) Pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_AggregateHandleCommand{
			AggregateHandleCommand: r,
		},
	}
}

// Pack returns a container that contains this record.
func (r *IntegrationHandleCommand) Pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_IntegrationHandleCommand{
			IntegrationHandleCommand: r,
		},
	}
}

// Pack returns a container that contains this record.
func (r *ProcessHandleEvent) Pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ProcessHandleEvent{
			ProcessHandleEvent: r,
		},
	}
}

// Pack returns a container that contains this record.
func (r *ProcessHandleTimeout) Pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ProcessHandleTimeout{
			ProcessHandleTimeout: r,
		},
	}
}

// Unpack returns the record contained in the container.
func (c *RecordContainer) Unpack() Record {
	return c.Elem.(element).unpack()
}

// element is an interface for the wrapper types used to store records in a
// record container.
type element interface {
	isRecordContainer_Elem
	unpack() Record
}

func (e *RecordContainer_ExecutorExecuteCommand) unpack() Record {
	return e.ExecutorExecuteCommand
}

func (e *RecordContainer_AggregateHandleCommand) unpack() Record {
	return e.AggregateHandleCommand
}

func (e *RecordContainer_IntegrationHandleCommand) unpack() Record {
	return e.IntegrationHandleCommand
}

func (e *RecordContainer_ProcessHandleEvent) unpack() Record {
	return e.ProcessHandleEvent
}

func (e *RecordContainer_ProcessHandleTimeout) unpack() Record {
	return e.ProcessHandleTimeout
}
