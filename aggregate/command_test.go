package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregate"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/onsi/gomega"
)

// executeCommandSync executes a command by sending it to a command channel and
// waiting for it to complete.
func executeCommandSync(
	ctx context.Context,
	commands chan<- *Command,
	command parcel.Parcel,
) {
	cmd := executeCommandAsync(ctx, commands, command)

	select {
	case <-ctx.Done():
		ExpectWithOffset(1, ctx.Err()).ShouldNot(HaveOccurred())
	case <-cmd.Done():
		ExpectWithOffset(1, cmd.Err()).ShouldNot(HaveOccurred())
	}
}

// executeCommandAsync executes a command by sending it to a command channel.
func executeCommandAsync(
	ctx context.Context,
	commands chan<- *Command,
	command parcel.Parcel,
) *Command {
	cmd := &Command{
		Context: ctx,
		Parcel:  command,
	}

	select {
	case <-ctx.Done():
		ExpectWithOffset(1, ctx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	return cmd
}

// commandAcknowledgerStub is a test implementation of the CommandAcknowledger
// interface.
type commandAcknowledgerStub struct {
	CommandAcknowledger

	PrepareAckFunc func(
		ctx context.Context,
		commandID string,
		hk, id string,
		rev uint64,
	) error

	CommitAckFunc func(
		ctx context.Context,
		commandID string,
		hk, id string,
		rev uint64,
	) error
}

func (s *commandAcknowledgerStub) PrepareAck(
	ctx context.Context,
	commandID string,
	hk, id string,
	rev uint64,
) error {
	if s.PrepareAckFunc != nil {
		return s.PrepareAckFunc(ctx, commandID, hk, id, rev)
	}

	if s.CommandAcknowledger != nil {
		return s.CommandAcknowledger.PrepareAck(ctx, commandID, hk, id, rev)
	}

	return nil
}

func (s *commandAcknowledgerStub) CommitAck(
	ctx context.Context,
	commandID string,
	hk, id string,
	rev uint64,
) error {
	if s.CommitAckFunc != nil {
		return s.CommitAckFunc(ctx, commandID, hk, id, rev)
	}

	if s.CommandAcknowledger != nil {
		return s.CommandAcknowledger.CommitAck(ctx, commandID, hk, id, rev)
	}

	return nil
}
