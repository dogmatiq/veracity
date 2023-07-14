package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/envelope"
)

type CommandExecutor struct {
	EnqueueCommands chan<- *EnqueueCommandExchange
	Packer          *envelope.Packer
}

func (e *CommandExecutor) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	done := make(chan error, 1)

	ex := &EnqueueCommandExchange{
		Command: e.Packer.Pack(c),
		Done:    done,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.EnqueueCommands <- ex:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
