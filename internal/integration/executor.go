package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/veracity/internal/messaging/ackqueue"
)

// CommandExecutor is an implementation of [dogma.CommandExecutor] that
// dispatches commands to a queue for execution.
type CommandExecutor struct {
	Commands *ackqueue.Queue[*envelopepb.Envelope]
	Packer   *envelopepb.Packer
}

// ExecuteCommand enqueues a command for execution.
func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	c dogma.Command,
	_ ...dogma.ExecuteCommandOption,
) error {
	return e.Commands.Push(
		ctx,
		e.Packer.Pack(c),
	)
}
