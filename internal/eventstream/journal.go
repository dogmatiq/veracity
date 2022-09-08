package eventstream

// GetEndOffset returns the offset AFTER the last event in the record.
func (x *AppendRecord) GetEndOffset() uint64 {
	return x.GetBeginOffset() + uint64(len(x.GetEnvelopes()))
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(s *EventStream)
}

func (x *JournalRecord_Append) apply(s *EventStream) { s.applyAppend(x.Append) }
