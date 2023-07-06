package test

import (
	"context"
	"time"
)

// ContextWithTimeout returns a context that is cancelled when the test completes.
func ContextWithTimeout(
	t TestingT,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx, cancel
}
