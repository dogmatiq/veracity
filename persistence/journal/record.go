package journal

import (
	"context"
)

// Record is an interface for a journal record.
type Record interface {
	Process(context.Context, Processor) error

	toElem() isContainer_Elem
}

// unpack unpacks a record from a container.
func (c *Container) unpack() Record {
	switch rec := c.Elem.(type) {
	case *Container_ExecutorExecuteCommand:
		return rec.ExecutorExecuteCommand
	case *Container_AggregateHandleCommand:
		return rec.AggregateHandleCommand
	case *Container_IntegrationHandleCommand:
		return rec.IntegrationHandleCommand
	case *Container_ProcessHandleEvent:
		return rec.ProcessHandleEvent
	case *Container_ProcessHandleTimeout:
		return rec.ProcessHandleTimeout
	default:
		panic("unrecognized record type")
	}
}

func (r *ExecutorExecuteCommand) toElem() isContainer_Elem {
	return &Container_ExecutorExecuteCommand{
		ExecutorExecuteCommand: r,
	}
}

func (r *AggregateHandleCommand) toElem() isContainer_Elem {
	return &Container_AggregateHandleCommand{
		AggregateHandleCommand: r,
	}
}

func (r *IntegrationHandleCommand) toElem() isContainer_Elem {
	return &Container_IntegrationHandleCommand{
		IntegrationHandleCommand: r,
	}
}

func (r *ProcessHandleEvent) toElem() isContainer_Elem {
	return &Container_ProcessHandleEvent{
		ProcessHandleEvent: r,
	}
}

func (r *ProcessHandleTimeout) toElem() isContainer_Elem {
	return &Container_ProcessHandleTimeout{
		ProcessHandleTimeout: r,
	}
}

// Processor is an interface for processing records in a journal.
type Processor interface {
	ProcessExecutorExecuteCommandRecord(context.Context, *ExecutorExecuteCommand) error
	ProcessAggregateHandleCommandRecord(context.Context, *AggregateHandleCommand) error
	ProcessIntegrationHandleCommandRecord(context.Context, *IntegrationHandleCommand) error
	ProcessProcessHandleEventRecord(context.Context, *ProcessHandleEvent) error
	ProcessProcessHandleTimeoutRecord(context.Context, *ProcessHandleTimeout) error
}

// Process calls p.ProcessExecutorExecuteCommandRecord(ctx, r).
func (r *ExecutorExecuteCommand) Process(ctx context.Context, p Processor) error {
	return p.ProcessExecutorExecuteCommandRecord(ctx, r)
}

// Process calls p.ProcessAggregateHandleCommandRecord(ctx, r).
func (r *AggregateHandleCommand) Process(ctx context.Context, p Processor) error {
	return p.ProcessAggregateHandleCommandRecord(ctx, r)
}

// Process calls p.ProcessIntegrationHandleCommandRecord(ctx, r).
func (r *IntegrationHandleCommand) Process(ctx context.Context, p Processor) error {
	return p.ProcessIntegrationHandleCommandRecord(ctx, r)
}

// Process calls p.ProcessProcessHandleEventRecord(ctx, r).
func (r *ProcessHandleEvent) Process(ctx context.Context, p Processor) error {
	return p.ProcessProcessHandleEventRecord(ctx, r)
}

// Process calls p.ProcessProcessHandleTimeoutRecord(ctx, r).
func (r *ProcessHandleTimeout) Process(ctx context.Context, p Processor) error {
	return p.ProcessProcessHandleTimeoutRecord(ctx, r)
}
