package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
)

type Loader struct {
	Snapshots    SnapshotLoader
	Transactions TransactionLoader
}

func (l *Loader) Load(
	ctx context.Context,
	handlerKey, instanceID string,
	new func() dogma.AggregateRoot,
) (dogma.AggregateRoot, uint64, error) {
	root := new()

	revision, err := l.Snapshots.LoadSnapshot(
		ctx,
		handlerKey,
		instanceID,
		root,
	)
	if err != nil {
		return nil, 0, err
	}

	for {
		transactions, done, err := l.Transactions.LoadTransactionsSinceRevision(
			ctx,
			handlerKey,
			instanceID,
			revision,
		)
		if err != nil {
			return nil, 0, err
		}

		for _, tx := range transactions {
			if tx.Destroyed {
				root = new()
			} else {
				for _, ev := range tx.Events {
					root.ApplyEvent(ev)
				}
			}

			revision++
		}

		if done {
			return root, revision, nil
		}
	}
}
