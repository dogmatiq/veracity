package aggregate

type journalRecord interface {
	isJournalRecord_OneOf
	apply(i *instance)
}

func (x *JournalRecord_Revision) apply(i *instance) { i.applyRevision(x.Revision) }
