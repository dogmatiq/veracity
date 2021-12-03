package main

import (
	"context"

	"github.com/dogmatiq/dapper"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	marshalfixtures "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/persistence/memory"
)

func main() {
	ctx := context.Background()

	committer := &journal.Committer{
		Journal: &memory.Journal{},
		Index:   &memory.KeyValueStore{},
	}

	m := fixtures.MessageA1

	env := &envelopespec.Envelope{
		MessageId:     "0001",
		CausationId:   "0001",
		CorrelationId: "0001",
		Description:   dogma.DescribeMessage(m),
	}

	marshalkit.MustMarshalMessageIntoEnvelope(
		marshalfixtures.Marshaler,
		m,
		env,
	)

	lastID, err := committer.SyncIndex(ctx)
	if err != nil {
		panic(err)
	}

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.ExecutorExecuteCommand{
			Envelope: env,
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)

	_, err = committer.Append(
		ctx,
		lastID,
		&journal.AggregateHandleCommand{
			MessageId: env.MessageId,
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)
}
