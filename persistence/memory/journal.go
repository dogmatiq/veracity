package memory

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/dogmatiq/veracity/persistence"
)

// Journal is an in-memory implementation of a persistence.Journal.
type Journal struct {
	recordsM sync.RWMutex
	records  [][]byte

	tailersM sync.Mutex
	tailers  []chan<- []byte
}

// Open returns a reader used to read journal records in order, beginning at
// the given record ID.
//
// If id is empty, the reader is opened at the first available record.
func (j *Journal) Open(ctx context.Context, id string) (persistence.Reader, error) {
	if id == "" {
		id = "0"
	}

	offset, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, errors.New("invalid record ID")
	}

	return &reader{
		journal: j,
		offset:  offset,
	}, nil
}

// Append adds a record to the end of the journal.
func (j *Journal) Append(ctx context.Context, rec []byte) (string, error) {
	j.recordsM.Lock()
	defer j.recordsM.Unlock()

	// Append the record.
	offset := uint64(len(j.records))
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

	return strconv.FormatUint(offset, 10), nil
}

// reader is an implementation of the persistence.Reader interface that reads
// from an in-memory journal.
type reader struct {
	journal    *Journal
	offset     uint64
	historical [][]byte
	realtime   <-chan []byte
}

// Next returns the next record in the journal or blocks until it
// becomes available.
func (r *reader) Next(ctx context.Context) (id string, data []byte, err error) {
	for {
		data, ok, err := r.waitNext(ctx)
		if err != nil {
			return "", nil, err
		}

		if !ok {
			data, ok = r.readHistorical()
		}

		if ok {
			offset := r.offset
			r.offset++

			return strconv.FormatUint(offset, 10), data, nil
		}
	}
}

func (r *reader) waitNext(ctx context.Context) ([]byte, bool, error) {
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

func (r *reader) readHistorical() ([]byte, bool) {
	if len(r.historical) == 0 {
		// We don't have a local copy of any historical records, attempt to
		// obtain more from the journal.
		r.journal.recordsM.RLock()
		defer r.journal.recordsM.RUnlock()

		r.historical = r.journal.records[r.offset:]

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

	data := r.historical[0]
	r.historical = r.historical[1:]

	return data, true
}

// Close closes the reader.
func (r *reader) Close() error {
	return nil
}
