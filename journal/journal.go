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
	// lastID is the ID of the last record that is expected to be in the
	// journal. If it does not match the ID of the actual last record, the
	// append operation fails.
	Append(ctx context.Context, lastID, data []byte) (id []byte, err error)

	// Scan reads the records in the journal in the order they were appended.
	//
	// If afterID is empty reading starts at the first record; otherwise,
	// reading starts at the record immediately after afterID.
	//
	// fn is called for each record until the end of the journal is reached, an
	// error occurs, or ctx is canceled.
	Scan(
		ctx context.Context,
		afterID []byte,
		fn func(ctx context.Context, id, data []byte) error,
	) (lastID []byte, err error)
}

// Append adds a record to the end of the journal.
//
// lastID is the ID of the last record that is expected to be in the journal. If
// it does not match the ID of the actual last record, the append operation
// fails.
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

// Scan visits the records in the journal in the order they were appended.
//
// If afterID is empty reading starts at the first record; otherwise, reading
// starts at the record immediately after afterID.
func Scan(
	ctx context.Context,
	j Journal,
	afterID []byte,
	v Visitor,
) (lastID []byte, err error) {
	return j.Scan(
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
