package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
)

type SnapshotLoader interface {
	LoadSnapshot(
		ctx context.Context,
		handlerKey, instanceID string,
		root dogma.AggregateRoot,
	) (uint64, error)
}

type SnapshotSaver interface {
	SaveSnapshot(
		ctx context.Context,
		handlerKey, instanceID string,
		revision uint64,
		root dogma.AggregateRoot,
	) error
}
