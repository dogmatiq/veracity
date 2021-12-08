package memory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/dogmatiq/veracity/persistence"
)

// Journal is an in-memory append-only sequence of opaque binary records.
//
// It is an implementation of the persistence.Journal interface intended for
// testing purposes.
type Journal struct {
	m          sync.RWMutex
	records    [][]byte
	nextOffset int
}

// Append adds a record to the end of the journal.
//
// prevID must be the ID of the most recent record, or an empty slice if the
// journal is currently empty; otherwise, the append operation fails.
func (j *Journal) Append(ctx context.Context, prevID, rec []byte) ([]byte, error) {
	prevOffset, err := recordIDToOffset(prevID)
	if err != nil {
		return nil, err
	}

	j.m.Lock()
	defer j.m.Unlock()

	if j.nextOffset-1 != prevOffset {
		return nil, fmt.Errorf(
			"optimistic lock failure, the last record ID is %#v but the caller provided %#v",
			string(offsetToRecordID(j.nextOffset-1)),
			string(prevID),
		)
	}

	j.records = append(j.records, rec)

	id := offsetToRecordID(j.nextOffset)
	j.nextOffset++

	return id, nil
}

// Truncate removes records from the beginning of the journal.
//
// keepID is the ID of the oldest record to keep. It becomes the record at
// the start of the journal.
func (j *Journal) Truncate(ctx context.Context, keepID []byte) error {
	keepOffset, err := recordIDToOffset(keepID)
	if err != nil {
		return err
	}

	j.m.Lock()
	defer j.m.Unlock()

	firstOffset := j.nextOffset - len(j.records)
	keepIndex := keepOffset - firstOffset

	if keepIndex < 0 {
		return errors.New("no such record")
	}

	j.records = j.records[keepIndex:]

	return nil
}

// Open returns a Reader that reads the records in the journal in the order they
// were appended.
//
// If afterID is empty reading starts at the first record; otherwise,
// reading starts at the record immediately after afterID.
func (j *Journal) Open(ctx context.Context, afterID []byte) (persistence.JournalReader, error) {
	offset, err := recordIDToOffset(afterID)
	if err != nil {
		return nil, err
	}

	return &reader{
		journal: j,
		offset:  offset + 1,
	}, nil
}

// reader is an implementation of persistence.JournalReader that read from an
// in-memory journal.
type reader struct {
	journal *Journal
	records [][]byte
	offset  int
}

// Next returns the next record in the journal.
//
// If ok is true, id is the ID of the next record and data is the record data
// itself.
//
// If ok is false the end of the journal has been reached. The reader should be
// closed and discarded; the behavior of subsequent calls to Next() is
// undefined.
func (r *reader) Next(ctx context.Context) (id, data []byte, ok bool, err error) {
	if len(r.records) == 0 {
		r.journal.m.RLock()
		records := r.journal.records
		nextOffset := r.journal.nextOffset
		r.journal.m.RUnlock()

		if r.offset == nextOffset {
			return nil, nil, false, nil
		}

		firstOffset := nextOffset - len(records)

		if r.offset < firstOffset {
			return nil, nil, false, fmt.Errorf(
				"record %#v has been truncated",
				string(offsetToRecordID(r.offset)),
			)
		}

		r.records = records[r.offset-firstOffset:]
	}

	id = offsetToRecordID(r.offset)
	r.offset++

	data = r.records[0]
	r.records = r.records[1:]

	return id, data, true, nil
}

// Close closes the reader.
func (r *reader) Close() error {
	return nil
}

// offsetToRecordID converts an offset to a record ID.
func offsetToRecordID(offset int) []byte {
	if offset == -1 {
		return nil
	}

	return []byte(strconv.Itoa(offset))
}

// recordIDToOffset converts a record ID to an offset.
func recordIDToOffset(id []byte) (int, error) {
	if len(id) == 0 {
		return -1, nil
	}

	offset, err := strconv.Atoi(string(id))
	if err != nil {
		return 0, fmt.Errorf("%#v is not a valid record ID", id)
	}

	return offset, nil
}
