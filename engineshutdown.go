package veracity

import (
	"context"
	"time"
)

const defaultShutdownTimeout = 15 * time.Second

// shutdownContextFor returns a context for shutting down an engine subsystem that
// has stopped because ctx is done.
//
// The returned context is NOT a child of ctx, as ctx is expected to have
// already been canceled.
func shutdownContextFor(ctx context.Context) (context.Context, context.CancelFunc) {
	// TODO: use ctx's cancel cause to obtain a shutdown context that is
	// controlled by the user.
	return context.WithTimeout(context.Background(), defaultShutdownTimeout)
}
