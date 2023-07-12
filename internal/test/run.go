package test

import (
	"context"
	"errors"
	"time"
)

// TaskRunner launches a task in the background.
type TaskRunner struct {
	t  TestingT
	fn func(ctx context.Context) error
}

// RunInBackground returns a [TaskRunner] that executes fn in its own goroutine.
func RunInBackground(
	t TestingT,
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

	ctx := contextOf(r.t)
	ctx, cancel := context.WithCancelCause(ctx)

	task := &Task{
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

// Done returns a channel that is closed when the function returns.
func (t *Task) Done() <-chan struct{} {
	return t.done
}

// Err returns the error returned by the function.
//
// It panics if the function has not yet returned.
func (t *Task) Err() error {
	select {
	case <-t.done:
		return t.err
	default:
		panic("background task has not returned")
	}
}

// ExpectCompletion waits for the background task to return successfully.
//
// It can be used after a call to t.Stop() to wait for the function to return.
func (t *Task) ExpectCompletion() {
	select {
	case <-t.done:
		if t.err != nil {
			t.t.Fatalf("background task returned an unexpected error: %s", t.err)
		}
	case <-time.After(shutdownTimeout):
		t.t.Fatalf("background task was canceled but did not return within %s", shutdownTimeout)
	}
}
