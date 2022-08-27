package aggregate

import (
	"context"
)

type Journal interface {
	Read(
		ctx context.Context,
		hk, id string,
		offset uint64,
	) ([]Record, error)

	Write(
		ctx context.Context,
		hk, id string,
		offset uint64,
		r Record,
	) error
}
