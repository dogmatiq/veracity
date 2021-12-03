package main

import (
	"context"

	"github.com/dogmatiq/dapper"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/persistence/memory"
	"google.golang.org/protobuf/encoding/prototext"
)

func main() {
	ctx := context.Background()

	committer := &journal.Committer{
		Journal:     &memory.Journal{},
		Index:       &memory.KeyValueStore{},
		Marshaler:   prototext.MarshalOptions{},
		Unmarshaler: prototext.UnmarshalOptions{},
	}

	lastID, err := committer.Sync(ctx)
	if err != nil {
		panic(err)
	}

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.ExecutorExecuteCommand{
			Envelope: &envelopespec.Envelope{
				MessageId: "0001",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.ExecutorExecuteCommand{
			Envelope: &envelopespec.Envelope{
				MessageId: "0002",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.ExecutorExecuteCommand{
			Envelope: &envelopespec.Envelope{
				MessageId: "0003",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.AggregateHandleCommand{
			MessageId: "0002",
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)

	lastID, err = committer.Append(
		ctx,
		lastID,
		&journal.AggregateHandleCommand{
			MessageId: "0001",
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
			MessageId: "0003",
		},
	)
	if err != nil {
		panic(err)
	}

	dapper.Print(committer.Index)
}
