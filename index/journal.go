package index

import (
	"context"

	"github.com/dogmatiq/veracity/index/internal/indexpb"
)

// journalKey is the key used to store the information about the journal itself.
var journalKey = makeKey("journal")

// loadJournal loads information about the journal from the index.
func loadJournal(ctx context.Context, index Index) (*indexpb.Journal, error) {
	j := &indexpb.Journal{}

	if err := getProto(ctx, index, journalKey, j); err != nil {
		return nil, err
	}

	return j, nil
}

// saveJournal loads information about the journal from the index.
func saveJournal(ctx context.Context, index Index, j *indexpb.Journal) error {
	return setProto(ctx, index, journalKey, j)
}
