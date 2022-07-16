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
	commands chan<- Command,
	command parcel.Parcel,
) {
	result := executeCommandAsync(ctx, commands, command)

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case err := <-result:
		Expect(err).ShouldNot(HaveOccurred())
	}
}

// executeCommandAsync executes a command by sending it to a command channel.
//
// It returns a channel that will receive the result of the command.
func executeCommandAsync(
	ctx context.Context,
	commands chan<- Command,
	command parcel.Parcel,
) <-chan error {
	result := make(chan error, 1)

	cmd := Command{
		Context: ctx,
		Parcel:  command,
		Result:  result,
	}

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	return result
}
