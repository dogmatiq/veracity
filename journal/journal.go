package journal

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// A Journal is an append-only immutable sequence of records.
//
// The journal is the engine's authoratative data store. All records must be
// kept indefinitely.
type Journal interface {
	// Append adds a record to the end of the journal.
	//
	// lastID must be the ID of the last record in the journal, or an empty
	// slice if the journal is currently empty; otherwise, the append operation
	// fails.
	Append(ctx context.Context, lastID, data []byte) (id []byte, err error)

	// OpenReader returns a reader that reads the records in the journal in the
	// order they were appended.
	//
	// If afterID is empty reading starts at the first record; otherwise,
	// reading starts at the record immediately after afterID.
	OpenReader(ctx context.Context, afterID []byte) (Reader, error)
}

// A Reader reads records from a journal in the order they were appended.
//
// Readers are not safe for concurrent use.
type Reader interface {
	// Next returns the next record in the journal.
	//
	// ok is false if there are no more records.
	Next(ctx context.Context) (id, data []byte, ok bool, err error)

	// Close closes the reader.
	Close() error
}

// Append adds a record to the end of the journal.
//
// lastID must be the ID of the last record in the journal, or an empty slice if
// the journal is currently empty; otherwise, the append operation fails.
func Append(
	ctx context.Context,
	j Journal,
	lastID []byte,
	rec Record,
) ([]byte, error) {
	data, err := proto.Marshal(&Container{
		Elem: rec.toElem(),
	})
	if err != nil {
		return nil, err
	}

	return j.Append(ctx, lastID, data)
}

// VisitRecords visits the records in the journal in the order they were
// appended.
//
// If afterID is empty reading starts at the first record; otherwise, reading
// starts at the record immediately after afterID.
func VisitRecords(
	ctx context.Context,
	j Journal,
	afterID []byte,
	v Visitor,
) (lastID []byte, err error) {
	r, err := j.OpenReader(ctx, afterID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	container := &Container{}

	for {
		id, data, ok, err := r.Next(ctx)
		if err != nil {
			return nil, err
		}

		if !ok {
			return lastID, nil
		}

		if err := proto.Unmarshal(data, container); err != nil {
			return nil, err
		}

		if err := container.Elem.(element).acceptVisitor(ctx, id, v); err != nil {
			return nil, err
		}

		lastID = id
	}
}
