package aggregate

import (
	"context"
	"sync"

	"github.com/dogmatiq/veracity/parcel"
)

// CommandAcknowledger acknowledges commands that have been handled
// successfully.
type CommandAcknowledger interface {
	PrepareAck(ctx context.Context, commandID string) error
	CommitAck(ctx context.Context, commandID string) error
}

// Command encapsulates a command message that is to be executed by a worker.
type Command struct {
	// Context is the context of the command itself.
	//
	// It can be used to set a deadline for handling and for propagating
	// command-scoped information such as tracing spans.
	Context context.Context

	// Parcel is the parcel containing the command message.
	Parcel parcel.Parcel

	m    sync.Mutex
	done chan struct{}
	err  error
}

// Ack is called to indicate that the command has been handled successfully.
func (c *Command) Ack() {
	c.m.Lock()
	defer c.m.Unlock()

	if c.done == nil {
		c.done = make(chan struct{})
	}

	close(c.done)
}

// Nack is called to indicate that an error occurred while handling the command.
func (c *Command) Nack(err error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.done == nil {
		c.done = make(chan struct{})
	}

	c.err = err

	close(c.done)
}

// Done returns a channel that is closed when the command is ACK'd or NACK'd.
func (c *Command) Done() <-chan struct{} {
	c.m.Lock()
	defer c.m.Unlock()

	if c.done == nil {
		c.done = make(chan struct{})
	}

	return c.done
}

// Err returns nil if the command was ACK'd, or the error that caused it to be
// NACK'd.
func (c *Command) Err() error {
	c.m.Lock()
	defer c.m.Unlock()

	return c.err
}
