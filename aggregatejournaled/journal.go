package aggregate

import (
	"context"
)

type JournalEntry interface {
	ApplyTo(*Snapshot)
}

type Journal interface {
	Read(
		ctx context.Context,
		hk, id string,
		offset uint64,
	) ([]JournalEntry, error)

	Write(
		ctx context.Context,
		hk, id string,
		offset uint64,
		e JournalEntry,
	) error
}
