package journal

// Record is an interface for a journal record.
type Record interface {
	// pack returns a container that contains this record.
	pack() *RecordContainer
}

// unpack returns the record contained in the container.
func (c *RecordContainer) unpack() Record {
	return c.Elem.(element).unpack()
}

// pack returns a container that contains this record.
func (r *ExecutorExecuteCommand) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ExecutorExecuteCommand{
			ExecutorExecuteCommand: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *AggregateHandleCommand) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_AggregateHandleCommand{
			AggregateHandleCommand: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *IntegrationHandleCommand) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_IntegrationHandleCommand{
			IntegrationHandleCommand: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *ProcessHandleEvent) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ProcessHandleEvent{
			ProcessHandleEvent: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *ProcessHandleTimeout) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_ProcessHandleTimeout{
			ProcessHandleTimeout: r,
		},
	}
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
