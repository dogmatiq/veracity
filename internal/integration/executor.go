package integration

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/messaging"
)

type CommandExecutor struct {
	ExecuteQueue *messaging.ExchangeQueue[ExecuteRequest, ExecuteResponse]
	Packer       *envelope.Packer
}

func (e *CommandExecutor) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	_, err := e.ExecuteQueue.Exchange(
		ctx, ExecuteRequest{
			Command:        e.Packer.Pack(c),
			IsFirstAttempt: true,
		},
	)
	return err
}
