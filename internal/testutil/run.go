package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/dogmatiq/veracity/internal/fsm"
)

// Run calls fn in its own goroutine.
//
// ctx is canceled when the test ends, at which point the function must return.
func Run(
	t *testing.T,
	fn func(ctx context.Context) error,
) (*fsm.Future[error], context.CancelFunc) {
	t.Helper()

	var err fsm.Future[error]

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		<-err.Ready()
	})

	go func() {
		err.Set(fn(ctx))
	}()

	return &err, cancel
}

// RunBeforeTestEnds calls fn in its own goroutine. fn must retun nil before the
// test ends, otherwise the test fails.
func RunBeforeTestEnds(
	t *testing.T,
	fn func(ctx context.Context) error,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	t.Cleanup(func() {
		select {
		case err := <-result:
			if err != nil {
				t.Errorf("unexpected error: %q", err)
			}
		default:
			t.Error("function did not return before the test completed")
		}

		cancel()
	})

	go func() {
		result <- fn(ctx)
	}()
}

// RunUntilTestEnds runs a function in its own goroutine for the entire duration
// of a test.
//
// ctx is canceled when the test ends, at which point the function must return
// [context.Canceled]. The test is marked as failed if the function returns
// before the test ends.
func RunUntilTestEnds(
	t *testing.T,
	run func(ctx context.Context) error,
) {
	t.Helper()

	var result fsm.Future[error]
	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		select {
		case <-result.Ready():
			t.Error("function returned before the test completed")
		default:
		}

		cancel()
		<-result.Ready()

		expect := context.Canceled
		if err := result.Get(); err != expect {
			t.Errorf("unexpected error: got %q, want %q", err, expect)
		}
	})

	go func() {
		result.Set(run(ctx))
	}()
}

// RunAfterDelay runs a function in its own goroutine after a delay.
//
// fn must retun nil before the test ends, otherwise the test fails.
func RunAfterDelay(
	t *testing.T,
	delay time.Duration,
	fn func(ctx context.Context) error,
) {
	t.Helper()

	RunBeforeTestEnds(
		t,
		func(ctx context.Context) error {
			select {
			case <-time.After(delay):
				return fn(ctx)
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)
}
