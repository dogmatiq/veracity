package memoryjournal

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

// Journal is an in-memory implementation of the journal.Journal interface.
type Journal struct {
	// records contains all records in the journal.
	//
	// record IDs are simply the string representation of the record's index,
	// converted to a byte slice.
	m       sync.RWMutex
	records [][]byte
}

// Append adds a record to the end of the journal.
//
// lastID is the ID of the last record in the journal. If it does not match the
// ID of the last record, the append operation fails.
func (j *Journal) Append(ctx context.Context, lastID, rec []byte) ([]byte, error) {
	lastIndex, err := recordIDToIndex(lastID)
	if err != nil {
		return nil, err
	}

	j.m.Lock()
	defer j.m.Unlock()

	n := len(j.records)

	if n-1 != lastIndex {
		return nil, fmt.Errorf(
			"optimistic lock failure, the last record ID is %#v but the caller provided %#v",
			string(indexToRecordID(n-1)),
			string(lastID),
		)
	}

	j.records = append(j.records, rec)

	return indexToRecordID(n), nil
}

// Read calls fn for each record in the journal, beginning at the record
// after the given record ID.
//
// If afterID is empty, reading starts at the first record.
func (j *Journal) Read(
	ctx context.Context,
	afterID []byte,
	fn func(ctx context.Context, id, data []byte) error,
) (lastID []byte, err error) {
	index, err := recordIDToIndex(afterID)
	if err != nil {
		return nil, err
	}

	var records [][]byte

	for {
		index++

		if index >= len(records) {
			j.m.RLock()
			records = j.records
			j.m.RUnlock()
		}

		if index >= len(records) {
			return indexToRecordID(index - 1), nil
		}

		id := indexToRecordID(index)

		if err := fn(ctx, id, records[index]); err != nil {
			return nil, err
		}
	}
}

// indexToRecordID converts an index to a record ID.
func indexToRecordID(index int) []byte {
	if index == -1 {
		return nil
	}

	return []byte(strconv.Itoa(index))
}

// recordID converts a record ID to an index.
func recordIDToIndex(id []byte) (int, error) {
	if len(id) == 0 {
		return -1, nil
	}

	index, err := strconv.Atoi(string(id))
	if err != nil {
		return 0, fmt.Errorf("%#v is not a valid record ID", id)
	}

	return index, nil
}
