package aggregate

import (
	"context"

	"github.com/dogmatiq/veracity/parcel"
)

type EventStream interface {
	Read(
		ctx context.Context,
		offset uint64,
	) ([]parcel.Parcel, error)

	Write(
		ctx context.Context,
		ev parcel.Parcel,
	) error
}
