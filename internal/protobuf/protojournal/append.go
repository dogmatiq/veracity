package protojournal

import (
	"context"
	"fmt"

	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// Append adds a record to the journal.
func Append[
	Record typedproto.Message[Struct],
	Struct typedproto.MessageStruct,
](
	ctx context.Context,
	j journal.Journal,
	end journal.Position,
	rec Record,
) error {
	data, err := typedproto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("unable to marshal record: %w", err)
	}

	return j.Append(ctx, end, data)
}
