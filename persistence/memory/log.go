package memory

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/dogmatiq/veracity/persistence"
)

// Log is an in-memory append-only sequence of immutable, opaque binary records.
//
// It is an implementation of the persistence.Log interface intended for testing
// purposes.
type Log struct {
	// records contains all records in the log.
	//
	// record IDs are simply the string representation of the record's index,
	// converted to a byte slice.
	m       sync.RWMutex
	records [][]byte
}

// Append adds a record to the end of the log.
//
// prevID must be the ID of the most recent record, or an empty slice if the
// log is currently empty; otherwise, the append operation fails.
func (l *Log) Append(ctx context.Context, prevID, rec []byte) ([]byte, error) {
	prevIndex, err := recordIDToIndex(prevID)
	if err != nil {
		return nil, err
	}

	l.m.Lock()
	defer l.m.Unlock()

	count := len(l.records)

	if count-1 != prevIndex {
		return nil, fmt.Errorf(
			"optimistic lock failure, the last record ID is %#v but the caller provided %#v",
			string(indexToRecordID(count-1)),
			string(prevID),
		)
	}

	l.records = append(l.records, rec)

	return indexToRecordID(count), nil
}

// Open returns a Reader that reads the records in the log in the order they
// were appended.
//
// If afterID is empty reading starts at the first record; otherwise,
// reading starts at the record immediately after afterID.
func (l *Log) Open(ctx context.Context, afterID []byte) (persistence.LogReader, error) {
	index, err := recordIDToIndex(afterID)
	if err != nil {
		return nil, err
	}

	return &reader{
		log:   l,
		index: index + 1,
	}, nil
}

// reader is an implementation of persistence.LogReader that read from an
// in-memory log.
type reader struct {
	log     *Log
	index   int
	records [][]byte
}

// Next returns the next record in the log.
//
// If ok is true, id is the ID of the next record and data is the record data
// itself.
//
// If ok is false the end of the log has been reached. The reader should be
// closed and discarded; the behavior of subsequent calls to Next() is
// undefined.
func (r *reader) Next(ctx context.Context) (id, data []byte, ok bool, err error) {
	if len(r.records) == 0 {
		r.log.m.RLock()
		r.records = r.log.records[r.index:]
		r.log.m.RUnlock()

		if len(r.records) == 0 {
			return nil, nil, false, nil
		}
	}

	id = indexToRecordID(r.index)
	r.index++

	data = r.records[0]
	r.records = r.records[1:]

	return id, data, true, nil
}

// Close closes the reader.
func (r *reader) Close() error {
	return nil
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
