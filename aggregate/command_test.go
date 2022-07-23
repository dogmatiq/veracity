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
	testCtx, cmdCtx context.Context,
	commands chan<- *Command,
	command parcel.Parcel,
) {
	cmd := executeCommandAsync(
		testCtx,
		cmdCtx,
		commands,
		command,
	)

	select {
	case <-testCtx.Done():
		ExpectWithOffset(1, testCtx.Err()).ShouldNot(HaveOccurred())
	case <-cmd.Done():
		ExpectWithOffset(1, cmd.Err()).ShouldNot(HaveOccurred())
	}
}

// executeCommandAsync executes a command by sending it to a command channel.
func executeCommandAsync(
	testCtx, cmdCtx context.Context,
	commands chan<- *Command,
	command parcel.Parcel,
) *Command {
	cmd := &Command{
		Context: cmdCtx,
		Parcel:  command,
	}

	select {
	case <-testCtx.Done():
		ExpectWithOffset(1, testCtx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	return cmd
}
