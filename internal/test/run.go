package test

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TaskRunner launches a task in the background.
type TaskRunner struct {
	t  *testing.T
	fn func(ctx context.Context) error
}

// RunInBackground returns a [TaskRunner] that executes fn in its own goroutine.
func RunInBackground(
	t *testing.T,
	fn func(ctx context.Context) error,
) TaskRunner {
	t.Helper()
	return TaskRunner{t, fn}
}

// UntilStopped executes the task in its own goroutine until the test ends or it
// is stopped explicitly.
func (r TaskRunner) UntilStopped() *Task {
	r.t.Helper()
	return r.run()
}

// BeforeTestEnds executes the task in its own goroutine before the test ends.
//
// If the task does not complete before the test ends, the test fails.
func (r TaskRunner) BeforeTestEnds() *Task {
	r.t.Helper()

	task := r.run()

	r.t.Cleanup(func() {
		r.t.Helper()

		select {
		case <-task.Done():
			if task.Err() != nil {
				r.t.Errorf("background task returned an unexpected error: %s", task.Err())
			}
		default:
			r.t.Error("background task did not return before the test ended")
		}
	})

	return task
}

// UntilTestEnds executes the task in its own goroutine until the test ends.
//
// If the task completes before the test ends, the test fails.
func (r TaskRunner) UntilTestEnds() *Task {
	r.t.Helper()

	task := r.run()

	r.t.Cleanup(func() {
		r.t.Helper()

		select {
		case <-task.Done():
			switch task.Err() {
			case errStopped:
				r.t.Log("background task was explicitly (but unexpectedly) stopped before the test ended")
			case nil:
				r.t.Error("background task returned before the test ended")
			default:
				r.t.Errorf("background task returned an error before the test ended: %s", task.Err())
			}
		default:
			task.Stop()
			<-task.Done()

			if task.Err() != errStopped {
				r.t.Errorf("background task returned an unexpected error: %s", task.Err())
			}
		}
	})

	return task
}

func (r TaskRunner) run() *Task {
	r.t.Helper()

	ctx, cancel := context.WithCancelCause(context.Background())

	task := &Task{
		t:      r.t,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		err := r.fn(ctx)

		if err == context.Canceled && ctx.Err() == context.Canceled {
			task.err = context.Cause(ctx)
		} else {
			task.err = err
		}

		close(task.done)
	}()

	r.t.Cleanup(func() {
		r.t.Helper()

		cancel(nil)

		select {
		case <-task.done:
		case <-time.After(shutdownTimeout):
			r.t.Errorf("background task's context was canceled but it did not return within %s", shutdownTimeout)
		}
	})

	return task
}

const shutdownTimeout = 10 * time.Second

var errStopped = errors.New("task stopped")

// Task represents a function running in the background.
type Task struct {
	t      TestingT
	cancel context.CancelCauseFunc
	done   chan struct{}
	err    error
}

// Stop cancels the context passed to the function.
func (t *Task) Stop() {
	t.cancel(errStopped)
}

// StopAndWait cancels the context passed to the function and waits for it to
// return.
//
// If it returns an error, the test fails.
func (t *Task) StopAndWait() {
	t.t.Helper()

	t.cancel(errStopped)

	select {
	case <-t.done:
		if t.err != errStopped {
			t.t.Fatalf("background task returned an unexpected error: %s", t.err)
		}
	case <-time.After(shutdownTimeout):
		t.t.Fatalf("background task was canceled but did not return within %s", shutdownTimeout)
	}
}

// Done returns a channel that is closed when the function returns.
func (t *Task) Done() <-chan struct{} {
	return t.done
}

// Err returns the error returned by the function.
//
// It panics if the function has not yet returned.
func (t *Task) Err() error {
	t.t.Helper()

	select {
	case <-t.done:
	default:
		t.t.Fatal("background task has not returned")
	}

	return t.err
}
