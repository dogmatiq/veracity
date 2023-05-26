package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
)

type CommandExecutor struct {
	Handler dogma.IntegrationMessageHandler
}

func (e *CommandExecutor) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	s := &scope{}

	return e.Handler.HandleCommand(ctx, s, c)
}
