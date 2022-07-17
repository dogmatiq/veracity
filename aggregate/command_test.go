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
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case <-cmd.Done():
		Expect(cmd.Err()).ShouldNot(HaveOccurred())
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
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	return cmd
}
