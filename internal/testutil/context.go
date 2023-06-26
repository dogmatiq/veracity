package testutil

import (
	"context"
	"testing"
	"time"
)

// ContextWithTimeout returns a context that is cancelled when the test completes.
func ContextWithTimeout(
	t *testing.T,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)

	return ctx, cancel
}
