package test

import (
	"context"
	"time"
)

// Run calls fn in its own goroutine.
//
// ctx is canceled when the test ends, at which point the function must return.
// The return value of fn does not affect the status of the test.
func Run(
	t TestingT,
	fn func(ctx context.Context) error,
) (err <-chan error, cancel context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	done := make(chan struct{})

	t.Cleanup(func() {
		cancel()
		<-done
	})

	go func() {
		result <- fn(ctx)
		close(done)
	}()

	return result, cancel
}

// RunUntilTestEnds calls fn in its own goroutine. If fn returns before the test
// ends then the test fails.
func RunUntilTestEnds(
	t TestingT,
	fn func(ctx context.Context) error,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	t.Cleanup(func() {
		var err error

		select {
		case err = <-result:
			t.Error("function returned before the test completed")
		default:
			cancel()
			err = <-result
		}

		if err != context.Canceled {
			t.Errorf("unexpected error: got %q, want %q", err, context.Canceled)
		}
	})

	go func() {
		result <- fn(ctx)
	}()
}

// CompleteBeforeTestEnds calls fn in its own goroutine. fn must return nil
// before the test ends, otherwise the test fails.
func CompleteBeforeTestEnds(
	t TestingT,
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

// CompleteAfterDelay calls fn in its own goroutine after a delay. fn must
// return nil before the test ends, otherwise the test fails.
func CompleteAfterDelay(
	t TestingT,
	delay time.Duration,
	fn func(ctx context.Context) error,
) {
	t.Helper()

	CompleteBeforeTestEnds(
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
