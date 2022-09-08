package queue

type journalRecord interface {
	isJournalRecord_OneOf
	load(q *Queue)
}

func (x *JournalRecord_Enqueue) load(q *Queue) { q.loadEnqueue(x.Enqueue) }
func (x *JournalRecord_Acquire) load(q *Queue) { q.loadAcquire(x.Acquire) }
func (x *JournalRecord_Ack) load(q *Queue)     { q.loadAck(x.Ack) }
func (x *JournalRecord_Reject) load(q *Queue)  { q.loadReject(x.Reject) }
