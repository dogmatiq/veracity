package aggregate

import (
	"context"

	"github.com/dogmatiq/veracity/parcel"
)

// Command encapsulates a message that is to be executed as a command.
type Command struct {
	Context context.Context
	Parcel  parcel.Parcel
	Result  chan<- error
}
