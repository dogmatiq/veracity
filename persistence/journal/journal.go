package journal

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// A Journal is an append-only immutable sequence of records.
type Journal interface {
	// Open returns a reader used to read journal records in order, beginning at
	// the record after the given record ID.
	//
	// If afterID is empty, the reader is opened at the first available record.
	Open(ctx context.Context, afterID []byte) (Reader, error)

	// LastID returns the ID of the last record in the journal.
	//
	// If the ID is empty the journal is empty.
	LastID(ctx context.Context) ([]byte, error)

	// Append adds a record to the end of the journal.
	//
	// lastID is the ID of the last record in the journal. If it does not match
	// the ID of the last record, the append operation fails.
	Append(ctx context.Context, lastID, data []byte) (id []byte, err error)
}

// A Reader is used to read journal record in order.
type Reader interface {
	// Next returns the next record in the journal or blocks until it becomes
	// available.
	Next(ctx context.Context) (id, data []byte, err error)

	// Close closes the reader.
	Close() error
}

// Append appends the binary representation of m to j.
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

// Read reads the next journal record into m.
func Read(ctx context.Context, r Reader) (id []byte, rec Record, err error) {
	id, data, err := r.Next(ctx)
	if err != nil {
		return nil, nil, err
	}

	c := &Container{}
	if err := proto.Unmarshal(data, c); err != nil {
		return nil, nil, err
	}

	return id, c.unpack(), nil
}
