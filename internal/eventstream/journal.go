package eventstream

// GetEndOffset returns the offset AFTER the last event in the record.
func (x *AppendRecord) GetEndOffset() uint64 {
	return x.GetBeginOffset() + uint64(len(x.GetEnvelopes()))
}
