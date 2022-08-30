package executor

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/veracity/parcel"
)

type Executor struct {
	Handler dogma.AggregateMessageHandler
	Journal Journal
	Ack     func(ctx context.Context, id string) error
}

func (e *Executor) ExecuteCommand(
	ctx context.Context,
	p parcel.Parcel,
) error {
	r := &fixtures.AggregateRoot{}
	s := &scope{root: r}
	e.Handler.HandleCommand(r, s, p.Message)

	return e.Ack(ctx, p.ID())
}
