package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/persistence/journal"
	"google.golang.org/protobuf/proto"
)

// Read reads a record from j and unmarshals it into rec.
func Read(
	ctx context.Context,
	j journal.Journal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, ok, err := j.Read(ctx, ver)
	if !ok || err != nil {
		return false, err
	}

	return true, proto.Unmarshal(data, rec)
}

// Append marshals rec to its binary representation and appends it to j.
func Append(
	ctx context.Context,
	j journal.Journal,
	ver uint64,
	rec proto.Message,
) (bool, error) {
	data, err := proto.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("unable to marshal journal record: %w", err)
	}

	return j.Append(ctx, ver, data)
}
