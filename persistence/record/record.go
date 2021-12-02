package record

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Processor is an interface for processing records in a journal.
type Processor interface {
	ProcessExecutorExecuteCommandRecord(context.Context, *ExecutorExecuteCommand) error
	ProcessAggregateHandleCommandRecord(context.Context, *AggregateHandleCommand) error
	ProcessIntegrationHandleCommandRecord(context.Context, *IntegrationHandleCommand) error
	ProcessProcessHandleEventRecord(context.Context, *ProcessHandleEvent) error
	ProcessProcessHandleTimeoutRecord(context.Context, *ProcessHandleTimeout) error
}

// Record is an interface for a journal record.
type Record interface {
	Process(context.Context, Processor) error

	wrap() isRecordContainer_Elem
}

func (c *RecordContainer) unwrap() Record {
	switch rec := c.Elem.(type) {
	case *RecordContainer_ExecutorExecuteCommand:
		return rec.ExecutorExecuteCommand
	case *RecordContainer_AggregateHandleCommand:
		return rec.AggregateHandleCommand
	case *RecordContainer_IntegrationHandleCommand:
		return rec.IntegrationHandleCommand
	case *RecordContainer_ProcessHandleEvent:
		return rec.ProcessHandleEvent
	case *RecordContainer_ProcessHandleTimeout:
		return rec.ProcessHandleTimeout
	default:
		panic("unrecognized record type")
	}
}

func (r *ExecutorExecuteCommand) wrap() isRecordContainer_Elem {
	return &RecordContainer_ExecutorExecuteCommand{
		ExecutorExecuteCommand: r,
	}
}

func (r *AggregateHandleCommand) wrap() isRecordContainer_Elem {
	return &RecordContainer_AggregateHandleCommand{
		AggregateHandleCommand: r,
	}
}

func (r *IntegrationHandleCommand) wrap() isRecordContainer_Elem {
	return &RecordContainer_IntegrationHandleCommand{
		IntegrationHandleCommand: r,
	}
}

func (r *ProcessHandleEvent) wrap() isRecordContainer_Elem {
	return &RecordContainer_ProcessHandleEvent{
		ProcessHandleEvent: r,
	}
}

func (r *ProcessHandleTimeout) wrap() isRecordContainer_Elem {
	return &RecordContainer_ProcessHandleTimeout{
		ProcessHandleTimeout: r,
	}
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

func Marshal(r Record) ([]byte, error) {
	return proto.Marshal(&RecordContainer{
		Elem: r.wrap(),
	})
}

func Unmarshal(data []byte) (Record, error) {
	var c RecordContainer
	if err := proto.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return c.unwrap(), nil
}
