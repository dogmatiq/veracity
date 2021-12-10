package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/veracity/discovery"
	"github.com/dogmatiq/veracity/persistence/memory"
	"golang.org/x/sync/errgroup"
)

type observer struct {
	nodeID string
	count  int
}

func (o *observer) PeerOnline(id string, data []byte) {
	o.count++
	fmt.Printf("node %s thinks peer %s is online, sees %d peers\n", o.nodeID, id, o.count)
}

func (o *observer) PeerOffline(id string) {
	o.count--
	fmt.Printf("node %s thinks peer %s is offline, sees %d peers\n", o.nodeID, id, o.count)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ctx := context.Background()
	journal := &memory.Journal{}

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 1000; i++ {
		nodeID := fmt.Sprintf("#%04d", i+1)
		a := discovery.Discoverer{
			NodeID:   nodeID,
			Observer: &observer{nodeID, 0},
			Journal:  journal,
			Logger: logging.Prefix(
				logging.DebugLogger,
				"%s: ",
				nodeID,
			),
		}

		g.Go(func() error {
			return a.Run(ctx)
		})
	}

	if err := g.Wait(); err != nil {
		panic(err)
	}
}

// func main() {

// 	committer := &journal.Committer{
// 		Journal:     &memory.Journal{},
// 		Index:       &memory.KeyValueStore{},
// 		Marshaler:   prototext.MarshalOptions{},
// 		Unmarshaler: prototext.UnmarshalOptions{},
// 	}

// 	lastID, err := committer.Sync(ctx)
// 	if err != nil {
// 		panic(err)
// 	}

// 	lastID, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandEnqueued{
// 			Envelope: &envelopespec.Envelope{
// 				MessageId: "0001",
// 			},
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)

// 	lastID, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandEnqueued{
// 			Envelope: &envelopespec.Envelope{
// 				MessageId: "0002",
// 			},
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)

// 	lastID, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandEnqueued{
// 			Envelope: &envelopespec.Envelope{
// 				MessageId: "0003",
// 			},
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)

// 	lastID, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandHandledByAggregate{
// 			MessageId: "0002",
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)

// 	lastID, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandHandledByAggregate{
// 			MessageId: "0001",
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)

// 	_, err = committer.Append(
// 		ctx,
// 		lastID,
// 		&journal.CommandHandledByAggregate{
// 			MessageId: "0003",
// 		},
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	dapper.Print(committer.Index)
// }
