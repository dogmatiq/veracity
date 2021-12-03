package logjournal

import (
	"context"

	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/persistence"
	"google.golang.org/protobuf/proto"
)

// Journal is an implementation of journal.Journal based on a persistence.Log.
type Journal struct {
	Log persistence.Log
}

// Append adds a record to the end of the journal.
//
// prevID must be the ID of the most recent record, or an empty slice if the
// journal is currently empty; otherwise, the append operation fails.
//
// It returns the ID of the newly appended record.
func (j *Journal) Append(ctx context.Context, prevID []byte, rec journal.Record) ([]byte, error) {
	data, err := proto.Marshal(rec.Pack())
	if err != nil {
		return nil, err
	}

	return j.Log.Append(ctx, prevID, data)
}

// Open returns a Reader that reads the records in the journal in the order
// they were appended.
//
// If afterID is empty reading starts at the first record; otherwise,
// reading starts at the record immediately after afterID.
func (j *Journal) Open(ctx context.Context, afterID []byte) (journal.Reader, error) {
	r, err := j.Log.Open(ctx, afterID)
	if err != nil {
		return nil, err
	}

	return &reader{reader: r}, nil
}

type reader struct {
	reader    persistence.LogReader
	container journal.RecordContainer
}

// Next returns the next record in the journal.
//
// If ok is true, id is the ID of the next record and rec is the record
// itself.
//
// If ok is false the end of the journal has been reached. The reader should
// be closed and discarded; the behavior of subsequent calls to Next() is
// undefined.
func (r *reader) Next(ctx context.Context) (id []byte, rec journal.Record, ok bool, err error) {
	id, data, ok, err := r.reader.Next(ctx)
	if !ok || err != nil {
		return nil, nil, ok, err
	}

	if err := proto.Unmarshal(data, &r.container); err != nil {
		return nil, nil, false, err
	}

	return id, r.container.Unpack(), ok, nil
}

// Close closes the reader.
func (r *reader) Close() error {
	return r.reader.Close()
}
