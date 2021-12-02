package journal

import (
	"context"
)

// Record is an interface for a journal record.
type Record interface {
	toElem() element
}

func (r *ExecutorExecuteCommand) toElem() element {
	return &Container_ExecutorExecuteCommand{
		ExecutorExecuteCommand: r,
	}
}

func (r *AggregateHandleCommand) toElem() element {
	return &Container_AggregateHandleCommand{
		AggregateHandleCommand: r,
	}
}

func (r *IntegrationHandleCommand) toElem() element {
	return &Container_IntegrationHandleCommand{
		IntegrationHandleCommand: r,
	}
}

func (r *ProcessHandleEvent) toElem() element {
	return &Container_ProcessHandleEvent{
		ProcessHandleEvent: r,
	}
}

func (r *ProcessHandleTimeout) toElem() element {
	return &Container_ProcessHandleTimeout{
		ProcessHandleTimeout: r,
	}
}

type element interface {
	isContainer_Elem
	acceptVisitor(context.Context, []byte, Visitor) error
}

func (e *Container_ExecutorExecuteCommand) acceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitExecutorExecuteCommandRecord(ctx, id, e.ExecutorExecuteCommand)
}

func (e *Container_AggregateHandleCommand) acceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitAggregateHandleCommandRecord(ctx, id, e.AggregateHandleCommand)
}

func (e *Container_IntegrationHandleCommand) acceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitIntegrationHandleCommandRecord(ctx, id, e.IntegrationHandleCommand)
}

func (e *Container_ProcessHandleEvent) acceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitProcessHandleEventRecord(ctx, id, e.ProcessHandleEvent)
}

func (e *Container_ProcessHandleTimeout) acceptVisitor(ctx context.Context, id []byte, v Visitor) error {
	return v.VisitProcessHandleTimeoutRecord(ctx, id, e.ProcessHandleTimeout)
}

// Visitor is an interface for processing records in a journal.
type Visitor interface {
	VisitExecutorExecuteCommandRecord(context.Context, []byte, *ExecutorExecuteCommand) error
	VisitAggregateHandleCommandRecord(context.Context, []byte, *AggregateHandleCommand) error
	VisitIntegrationHandleCommandRecord(context.Context, []byte, *IntegrationHandleCommand) error
	VisitProcessHandleEventRecord(context.Context, []byte, *ProcessHandleEvent) error
	VisitProcessHandleTimeoutRecord(context.Context, []byte, *ProcessHandleTimeout) error
}
