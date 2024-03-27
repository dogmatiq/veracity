package integrationjournal

import (
	"context"

	uuidpb "github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/persistencekit/marshaler"
)

// Open returns a journal that contains the state of the integration handler
// with the given key.
func Open(
	ctx context.Context,
	s journal.BinaryStore,
	key *uuidpb.UUID,
) (journal.Journal[*Record], error) {
	store := journal.NewMarshalingStore(s, recordMarshaler)
	name := Name(key)
	return store.Open(ctx, name)
}

var recordMarshaler = marshaler.NewProto[*Record]()

// Name returns the name of the journal that contains the state of the
// integration handler with the given key.
func Name(key *uuidpb.UUID) string {
	return "integration:" + key.AsString()
}
