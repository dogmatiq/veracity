package memoryjournal

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/dogmatiq/veracity/persistence/journal"
)

// Journal is an in-memory implementation of a persistence.Journal.
type Journal struct {
	// records contains all records in the journal.
	//
	// record IDs are simply the string representation of the record's index,
	// converted to a byte slice.
	recordsM sync.RWMutex
	records  [][]byte

	// tailers is a set of channels belong to readers that are "tailing" the
	// journal, meaning that they have read all historical records are are now
	// reading records in "real-time".
	tailersM sync.Mutex
	tailers  []chan<- []byte
}

// Open returns a reader used to read journal records in order, beginning at
// the record after the given record ID.
//
// If afterID is empty, the reader is opened at the first available record.
func (j *Journal) Open(ctx context.Context, afterID []byte) (journal.Reader, error) {
	r := &reader{
		journal: j,
	}

	if len(afterID) != 0 {
		index, err := recordIDToIndex(afterID)
		if err != nil {
			return nil, err
		}

		r.index = index + 1
	}

	return r, nil
}

// LastID returns the ID of the last record in the journal.
//
// If the ID is empty the journal is empty.
func (j *Journal) LastID(ctx context.Context) ([]byte, error) {
	j.recordsM.RLock()
	defer j.recordsM.RUnlock()

	return j.lastID(), nil
}

// lastID returns the ID of the last record in the journal.
//
// It expects j.recordsM to be locked.
func (j *Journal) lastID() []byte {
	n := len(j.records)

	if n == 0 {
		return nil
	}

	return indexToRecordID(n - 1)
}

// Append adds a record to the end of the journal.
//
// lastID is the ID of the last record in the journal. If it does not match the
// ID of the last record, the append operation fails.
func (j *Journal) Append(ctx context.Context, lastID, rec []byte) ([]byte, error) {
	j.recordsM.Lock()
	defer j.recordsM.Unlock()

	if expect := j.lastID(); !bytes.Equal(lastID, expect) {
		return nil, fmt.Errorf(
			"last ID does not match (expected %#v, got %#v)",
			string(expect),
			string(lastID),
		)
	}

	// Only now that lastID has been validated can we append the record.
	j.records = append(j.records, rec)
	j.notifyReaders(rec)

	return j.lastID(), nil
}

// Notify any readers that are "tailing" the journal of a new record.
func (j *Journal) notifyReaders(rec []byte) {
	j.tailersM.Lock()
	defer j.tailersM.Unlock()

	i := 0
	n := len(j.tailers)

	for i < n {
		ch := j.tailers[i]

		select {
		case ch <- rec:
			// We managed to notify the reader, move on to the next one.
			i++
		default:
			// This reader's buffer is full, remove it from the list of tailers
			// and close the channel so that it falls back to reading records
			// "historically".
			close(ch)
			j.tailers[i] = j.tailers[n-1] // move last entry to this index
			j.tailers[n-1] = nil          // clear entry at last index to prevent memory leak
			n--                           // adjust the known length of the slice
			j.tailers = j.tailers[:n]     // shrink the slice
		}
	}
}

// reader is an implementation of the journal.Reader interface that reads from
// an in-memory journal.
type reader struct {
	// journal is the the Journal that from which the Reader gets its records.
	journal *Journal

	// index is the index into the records that the reader will return next.
	index int

	// historical is a copy of a subset of the journals records, starting at the
	// index above. This is used to allow fast reads without locking the
	// journal.
	historical [][]byte

	// realtime is channel used to obtain records from the journal in
	// "real-time". It is only created when the list of historical events is
	// entirely exhausted.
	realtime <-chan []byte
}

// Next returns the next record in the journal or blocks until it becomes
// available.
func (r *reader) Next(ctx context.Context) (id, data []byte, err error) {
	for {
		data, ok, err := r.waitNext(ctx)
		if err != nil {
			return nil, nil, err
		}

		if !ok {
			data, ok = r.readHistorical()
		}

		if ok {
			id := indexToRecordID(r.index)
			r.index++
			return id, data, nil
		}
	}
}

// waitNext waits until the next message is appended to the journal.
//
// If ok is false, then r.realtime has been closed and the reader should attempt
// to catch up by reading historical records.
func (r *reader) waitNext(ctx context.Context) (data []byte, ok bool, err error) {
	// We are already "tailing" the journal. We can read directly from the
	// channel to get records in "real-time".
	if r.realtime != nil {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()

		case data, ok := <-r.realtime:
			if ok {
				return data, true, nil
			}

			// The channel was closed, meaning that this reader was unable
			// to keep up with the velocity of new records, we fall-back to
			// reading historical records.
			r.realtime = nil
		}
	}

	return nil, false, nil
}

// readHistorical reads the next historical record from the journal.
func (r *reader) readHistorical() (data []byte, ok bool) {
	if len(r.historical) == 0 {
		// We don't have a local copy of any historical records, attempt to
		// obtain more from the journal.
		r.journal.recordsM.RLock()
		defer r.journal.recordsM.RUnlock()

		r.historical = r.journal.records[r.index:]

		if len(r.historical) == 0 {
			// There are no more historical records to read, so we need to start
			// "tailing" the journal using a "real-time" channel.
			ch := make(chan []byte, 10) // TODO: adjust buffer size?
			r.realtime = ch

			r.journal.tailersM.Lock()
			defer r.journal.tailersM.Unlock()

			r.journal.tailers = append(r.journal.tailers, ch)

			return nil, false
		}
	}

	// Pop the first record from the historical list and return it.
	data = r.historical[0]
	r.historical = r.historical[1:]

	return data, true
}

// Close closes the reader.
func (r *reader) Close() error {
	return nil
}

// indexToRecordID converts an index to a record ID.
func indexToRecordID(index int) []byte {
	return []byte(strconv.Itoa(index))
}

// recordID converts a record ID to an index.
func recordIDToIndex(id []byte) (int, error) {
	index, err := strconv.Atoi(string(id))
	if err != nil {
		return 0, fmt.Errorf("%#v is not a valid record ID", id)
	}

	return index, nil
}
