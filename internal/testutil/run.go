package testutil

import (
	"context"
	"testing"

	"github.com/dogmatiq/veracity/internal/future"
)

// Go calls run in its own goroutine.
//
// It calls the run function with a context that is canceled when the test ends.
// It returns a future value that is resolved with the return value of run.
func Go(
	t *testing.T,
	run func(ctx context.Context) error,
) future.Future[error] {
	t.Helper()

	err, resolver := future.New[error]()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		<-err.Ready()
	})

	go func() {
		err := run(ctx)
		resolver.Set(err)
	}()

	return err
}
