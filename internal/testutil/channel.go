package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/dogmatiq/veracity/internal/fsm"
)

// Send executes ch <- v, or fails if the send blocks for more than
// 10 seconds.
func Send[T any](t *testing.T, ch chan<- T, v T) {
	t.Helper()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting to send value")
	case ch <- v:
	}
}

// RecvWithTimeout executes v, ok := <-ch, or fails if the receive blocks for
// more than 10 seconds.
func RecvWithTimeout[T any](t *testing.T, ch <-chan T) (v T, ok bool) {
	t.Helper()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting to receive value")
	case v, ok = <-ch:
	}

	return v, ok
}

func Exchange[Q, S any](
	t *testing.T,
	q *fsm.RequestQueue[Q, S],
	req Q,
) S {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := q.Exchange(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	return res
}
