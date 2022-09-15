package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/journal"
	"google.golang.org/protobuf/proto"
)

// Read reads a record from j and unmarshals it into rec.
func Read(
	ctx context.Context,
	j journal.BinaryJournal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, ok, err := j.Read(ctx, ver)
	if !ok || err != nil {
		return false, err
	}

	return true, proto.Unmarshal(data, rec)
}

// Write marshals rec to its binary representation and writes it to j.
func Write(
	ctx context.Context,
	j journal.BinaryJournal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, err := proto.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("unable to marshal journal record: %w", err)
	}

	return j.Write(ctx, ver, data)
}
