package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/messaging"
)

// CommandExecutor is an implementation of [dogma.CommandExecutor] that
// dispatches the command to an exchange queue.
type CommandExecutor struct {
	ExecuteQueue *messaging.ExchangeQueue[ExecuteRequest, ExecuteResponse]
	Packer       *envelope.Packer
}

// ExecuteCommand enqueues a command.
func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	c dogma.Command,
	_ ...dogma.ExecuteCommandOption,
) error {
	_, err := e.ExecuteQueue.Do(
		ctx, ExecuteRequest{
			Command: e.Packer.Pack(c),
		},
	)
	return err
}
