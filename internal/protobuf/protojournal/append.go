package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
)

// Append adds a record to the journal.
func Append[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.BinaryJournal,
	end journal.Position,
	rec Record,
) error {
	data, err := typedproto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("unable to marshal record: %w", err)
	}

	return j.Append(ctx, end, data)
}
