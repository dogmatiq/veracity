package router

import (
	"context"

	"github.com/dogmatiq/dogma"
)

type Router struct {
}

func (r *Router) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	return nil
}
