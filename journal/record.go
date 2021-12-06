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
func (r *CommandEnqueued) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_CommandEnqueued{
			CommandEnqueued: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *CommandHandledByAggregate) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_CommandHandledByAggregate{
			CommandHandledByAggregate: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *CommandHandledByIntegration) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_CommandHandledByIntegration{
			CommandHandledByIntegration: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *EventHandledByProcess) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_EventHandledByProcess{
			EventHandledByProcess: r,
		},
	}
}

// pack returns a container that contains this record.
func (r *TimeoutHandledByProcess) pack() *RecordContainer {
	return &RecordContainer{
		Elem: &RecordContainer_TimeoutHandledByProcess{
			TimeoutHandledByProcess: r,
		},
	}
}

// element is an interface for the wrapper types used to store records in a
// record container.
type element interface {
	isRecordContainer_Elem
	unpack() Record
}

func (e *RecordContainer_CommandEnqueued) unpack() Record {
	return e.CommandEnqueued
}

func (e *RecordContainer_CommandHandledByAggregate) unpack() Record {
	return e.CommandHandledByAggregate
}

func (e *RecordContainer_CommandHandledByIntegration) unpack() Record {
	return e.CommandHandledByIntegration
}

func (e *RecordContainer_EventHandledByProcess) unpack() Record {
	return e.EventHandledByProcess
}

func (e *RecordContainer_TimeoutHandledByProcess) unpack() Record {
	return e.TimeoutHandledByProcess
}
