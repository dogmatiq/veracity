package aggregate

import (
	"context"
)

type SnapshotStore interface {
	Read(ctx context.Context, hk, id string) (Snapshot, error)
	Write(ctx context.Context, hk, id string, sn Snapshot) error
}
