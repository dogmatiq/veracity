package journal

import (
	"context"
	"sync"
)

// MemoryJournal is an in-memory implementation of a journal.
type MemoryJournal struct {
	recordsM sync.RWMutex
	records  [][]byte

	tailersM sync.Mutex
	tailers  []chan<- []byte
}

// Open returns a reader used to read journal records in order, beginning at
// the given offset.
func (j *MemoryJournal) Open(ctx context.Context, offset uint64) (Reader, error) {
	return &memoryReader{
		journal: j,
		offset:  offset,
	}, nil
}

// Append adds a record to the end of the journal.
func (j *MemoryJournal) Append(ctx context.Context, rec []byte) (uint64, error) {
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

	return offset, nil
}

// memoryReader is an implementation of the Reader interface that reads from a
// MemoryJournal.
type memoryReader struct {
	journal    *MemoryJournal
	offset     uint64
	historical [][]byte
	realtime   <-chan []byte
}

// Next returns the next record in the journal.
//
// It blocks until a record becomes available, an error occurs, or ctx is
// canceled.
func (r *memoryReader) Next(ctx context.Context) (uint64, []byte, error) {
	rec, err := r.next(ctx)
	if err != nil {
		return 0, nil, err
	}

	offset := r.offset
	r.offset++
	return offset, rec, nil
}

func (r *memoryReader) next(ctx context.Context) ([]byte, error) {
	for {
		// We are already "tailing" the journal. We can read directly from the
		// channel to get records in "real-time".
		if r.realtime != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case rec, ok := <-r.realtime:
				if ok {
					return rec, nil
				}

				// The channel was closed, meaning that this reader was unable
				// to keep up with the velocity of new records, we fall-back to
				// reading historical records.
				r.realtime = nil
			}
		}

		// Return an historical record if one is available.
		if len(r.historical) > 0 {
			rec := r.historical[0]
			r.historical = r.historical[1:]
			return rec, nil
		}

		// Otherwise, check if there are any more historical records in the
		// journal that we can copy.
		r.journal.recordsM.RLock()
		r.historical = r.journal.records[r.offset:]

		if len(r.historical) == 0 {
			// There are no more historical records to read, so we need to start
			// "tailing" the journal using a "real-time" channel.
			ch := make(chan []byte, 10) // TODO: adjust buffer size?
			r.realtime = ch

			r.journal.tailersM.Lock()
			r.journal.tailers = append(r.journal.tailers, ch)
			r.journal.tailersM.Unlock()
		}

		r.journal.recordsM.RUnlock()
	}
}

// Close closes the reader.
func (r *memoryReader) Close() error {
	return nil
}
