package main

import (
	"context"

	"github.com/dogmatiq/dapper"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	marshalfixtures "github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/index"
	"github.com/dogmatiq/veracity/index/memoryindex"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/memoryjournal"
)

func main() {
	ctx := context.Background()

	jrn := &memoryjournal.Journal{}
	idx := &memoryindex.Index{}

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

	lastID, err := journal.Append(
		ctx,
		jrn,
		nil,
		&journal.ExecutorExecuteCommand{
			Envelope: env,
		},
	)
	if err != nil {
		panic(err)
	}

	b := &index.Builder{
		Journal: jrn,
		Index:   idx,
	}

	if _, err := b.Build(ctx); err != nil {
		panic(err)
	}

	dapper.Print(idx)

	_, err = journal.Append(
		ctx,
		jrn,
		lastID,
		&journal.AggregateHandleCommand{
			MessageId: env.MessageId,
		},
	)
	if err != nil {
		panic(err)
	}

	if _, err := b.Build(ctx); err != nil {
		panic(err)
	}

	dapper.Print(idx)
}
