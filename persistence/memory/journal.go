package memory

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/dogmatiq/veracity/persistence"
)

// Journal is an in-memory implementation of a persistence.Journal.
type Journal struct {
	// records contains all records in the journal.
	// record IDs are simply the record's index represented as a string.
	recordsM sync.RWMutex
	records  [][]byte

	// tailers is a set of channels belong to readers that are "tailing" the
	// journal, meaning that they have read all historical records are are now
	// reading records in "real-time".
	tailersM sync.Mutex
	tailers  []chan<- []byte
}

// Open returns a reader used to read journal records in order, beginning at
// the given record ID.
//
// If id is empty, the reader is opened at the first available record.
func (j *Journal) Open(ctx context.Context, id string) (persistence.JournalReader, error) {
	r := &reader{
		journal: j,
	}

	if id != "" {
		var err error
		r.index, err = strconv.ParseUint(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%#v is not a valid record ID", id)
		}
	}

	return r, nil
}

// Append adds a record to the end of the journal.
//
// lastID is the ID of the last record known to be in the journal. If it
// does not match the ID of the last record, the append operation fails.
func (j *Journal) Append(ctx context.Context, lastID string, rec []byte) (string, error) {
	j.recordsM.Lock()
	defer j.recordsM.Unlock()

	// Determine the index of the new record.
	index := uint64(len(j.records))

	// Verify that lastID is matches the actual last record.
	var expectedLastID string
	if index == 0 {
		expectedLastID = ""
	} else {
		expectedLastID = strconv.FormatUint(index-1, 10)
	}

	if lastID != expectedLastID {
		return "", fmt.Errorf(
			"last ID does not match (expected %#v, got %#v)",
			expectedLastID,
			lastID,
		)
	}

	// Only now that lastID has been validated can we append the record.
	j.records = append(j.records, rec)

	// Notify any readers that are "tailing" the journal of the new record.
	//
	// There's no need to lock tailersM, as it is only ever locked by goroutines
	// that already hold a lock on recordsM, which we currently have locked
	// exclusively.
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

	return strconv.FormatUint(index, 10), nil
}

// reader is an implementation of the persistence.Reader interface that reads
// from an in-memory journal.
type reader struct {
	journal    *Journal
	index      uint64
	historical [][]byte
	realtime   <-chan []byte
}

// Next returns the next record in the journal or blocks until it becomes
// available.
//
// If more is true there are guaranteed to be additional records in the journal
// after the one returned. A value of false does NOT guarantee there are no
// additional records.
func (r *reader) Next(ctx context.Context) (id string, data []byte, more bool, err error) {
	for {
		data, ok, more, err := r.waitNext(ctx)
		if err != nil {
			return "", nil, false, err
		}

		if !ok {
			data, ok, more = r.readHistorical()
		}

		if ok {
			id := strconv.FormatUint(r.index, 10)
			r.index++
			return id, data, more, nil
		}
	}
}

func (r *reader) waitNext(ctx context.Context) (data []byte, ok, more bool, err error) {
	// We are already "tailing" the journal. We can read directly from the
	// channel to get records in "real-time".
	if r.realtime != nil {
		select {
		case <-ctx.Done():
			return nil, false, false, ctx.Err()

		case data, ok := <-r.realtime:
			if ok {
				return data, true, len(r.realtime) > 0, nil
			}

			// The channel was closed, meaning that this reader was unable
			// to keep up with the velocity of new records, we fall-back to
			// reading historical records.
			r.realtime = nil
		}
	}

	return nil, false, false, nil
}

func (r *reader) readHistorical() (data []byte, ok, more bool) {
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

			return nil, false, false
		}
	}

	data = r.historical[0]
	r.historical = r.historical[1:]

	return data, true, len(r.historical) > 0
}

// Close closes the reader.
func (r *reader) Close() error {
	return nil
}
