package journal

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// A Journal is an append-only immutable sequence of records.
type Journal interface {
	// Append adds a record to the end of the journal.
	//
	// lastID is the ID of the last record in the journal. If it does not match
	// the ID of the last record, the append operation fails.
	Append(ctx context.Context, lastID, data []byte) (id []byte, err error)

	// Read calls fn for each record in the journal, beginning at the record
	// after the given record ID.
	//
	// If afterID is empty, reading starts at the first record.
	Read(
		ctx context.Context,
		afterID []byte,
		fn func(ctx context.Context, id, data []byte) error,
	) (lastID []byte, err error)
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
func Read(
	ctx context.Context,
	j Journal,
	afterID []byte,
	v Visitor,
) (lastID []byte, err error) {
	return j.Read(
		ctx,
		afterID,
		func(ctx context.Context, id, data []byte) error {
			c := &Container{}
			if err := proto.Unmarshal(data, c); err != nil {
				return err
			}

			return c.Elem.(element).acceptVisitor(ctx, id, v)
		},
	)
}
