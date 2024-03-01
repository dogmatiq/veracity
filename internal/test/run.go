package test

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TaskRunner launches a task in the background.
type TaskRunner struct {
	t    *testing.T
	name string
	fn   func(ctx context.Context) error
}

// RunInBackground returns a [TaskRunner] that executes fn in its own goroutine.
func RunInBackground(
	t *testing.T,
	name string,
	fn func(ctx context.Context) error,
) TaskRunner {
	return TaskRunner{t, name, fn}
}

// UntilStopped executes the task in its own goroutine with the expectation that
// it will not return until it is stopped explicitly, or the test ends.
func (r TaskRunner) UntilStopped() *Task {
	r.t.Helper()
	r.t.Logf("running %q in the background until it is stopped explicitly", r.name)

	return r.run(
		r.fn,
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				// expected
			case errStopped:
				// expected
			case nil:
				r.t.Errorf("%q returned successfully, expected it to be stopped explicitly", r.name)
			default:
				r.t.Errorf("%q returned an error, expected it to be stopped explicitly: %s", r.name, err)
			}
		},
	)
}

// RepeatedlyUntilStopped executes the task in its own goroutine, restarting it
// if it returns before it is stopped explicitly or the test ends.
func (r TaskRunner) RepeatedlyUntilStopped() *Task {
	r.t.Helper()
	r.t.Logf("repeatedly running %q in the background until it is stopped explicitly", r.name)

	return r.run(
		func(ctx context.Context) error {
			for {
				err := r.fn(ctx)

				if ctx.Err() == context.Canceled {
					return err
				} else if err == nil {
					r.t.Logf("restarting %q because it returned successfully before being stopped", r.name)
				} else {
					r.t.Logf("restarting %q because it returned an error: %s", r.name, err)
				}
			}
		},
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				// expected
			case errStopped:
				// expected
			case nil:
				r.t.Errorf("%q returned successfully, expected it to be stopped explicitly", r.name)
			default:
				r.t.Errorf("%q returned an error, expected it to be stopped explicitly: %s", r.name, err)
			}
		},
	)
}

// RepeatedlyUntilSuccess executes the task in its own goroutine, restarting it
// if it returns an error before the test ends.
func (r TaskRunner) RepeatedlyUntilSuccess() *Task {
	r.t.Helper()
	r.t.Logf("repeatedly running %q in the background until it returns successfully", r.name)

	return r.run(
		func(ctx context.Context) error {
			for {
				err := r.fn(ctx)

				if ctx.Err() == context.Canceled {
					return err
				} else if err == nil {
					return nil
				}
				r.t.Logf("restarting %q because it returned an error: %s", r.name, err)
			}
		},
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				// expected
			case errStopped:
				r.t.Errorf("%q was stopped explicitly, expected it to return successfully", r.name)
			case nil:
				// expected
			default:
				r.t.Errorf("%q returned an error, expected it to return successfully: %s", r.name, err)
			}
		},
	)
}

// BeforeTestEnds executes the task in its own goroutine with the expectation
// that it will return successfully before the test ends.
func (r TaskRunner) BeforeTestEnds() *Task {
	r.t.Helper()
	r.t.Logf("running %q to completion in the background before the test ends", r.name)

	return r.run(
		r.fn,
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				r.t.Errorf("%q did not return before the test ended, expected it to return successfully", r.name)
			case errStopped:
				r.t.Errorf("%q was stopped explicitly, expected it to return successfully", r.name)
			case nil:
				// expected
			default:
				r.t.Errorf("%q returned an error, expected it to return successfully: %s", r.name, err)
			}
		},
	)
}

// FailBeforeTestEnds executes the task in its own goroutine with the
// expectation that it will return an error before the test ends.
func (r TaskRunner) FailBeforeTestEnds() *Task {
	r.t.Helper()
	r.t.Logf("running %q to failure in the background before the test ends", r.name)

	return r.run(
		r.fn,
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				r.t.Errorf("%q did not return before the test ended, expected it to return an error", r.name)
			case errStopped:
				r.t.Errorf("%q was stopped explicitly, expected it to return an error", r.name)
			case nil:
				r.t.Errorf("%q returned successfully, expected it to return an error", r.name)
			default:
				// expected
			}
		},
	)
}

// UntilTestEnds executes the task in its own goroutine with the expectation
// that it will not return before the test ends.
func (r TaskRunner) UntilTestEnds() *Task {
	r.t.Helper()
	r.t.Logf("running %q in the background background until the test ends", r.name)

	return r.run(
		r.fn,
		func(err error) {
			r.t.Helper()

			switch err {
			case errTestEnded:
				// expected
			case errStopped:
				r.t.Errorf("%q was stopped explicitly, expected it to run until the test ended", r.name)
			case nil:
				r.t.Errorf("%q returned successfully, expected it to run until the test ended", r.name)
			default:
				r.t.Errorf("%q returned an error, expected it to run until the test ended: %s", r.name, err)
			}
		},
	)
}

func (r TaskRunner) run(
	fn func(ctx context.Context) error,
	expect func(err error),
) *Task {
	r.t.Helper()

	ctx, cancel := context.WithCancelCause(context.Background())

	task := &Task{
		t:      r.t,
		name:   r.name,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		r.t.Helper()

		task.err = fn(ctx)

		if task.err == context.Canceled && ctx.Err() == context.Canceled {
			task.err = context.Cause(ctx)
		}

		switch task.err {
		case errTestEnded, errStopped:
		case nil:
			r.t.Logf("%q returned successfully", r.name)
		default:
			r.t.Logf("%q returned an error: %s", r.name, task.err)
		}

		close(task.done)
	}()

	r.t.Cleanup(func() {
		r.t.Helper()

		cancel(errTestEnded)

		select {
		case <-time.After(shutdownTimeout):
			r.t.Errorf("%q did not return within %s of being stopped", r.name, shutdownTimeout)
		case <-task.done:
			expect(task.err)
		}
	})

	return task
}

const shutdownTimeout = 10 * time.Second

var (
	errTestEnded = errors.New("stopped by test")
	errStopped   = errors.New("stopped explicitly")
)

// Task represents a function running in the background.
type Task struct {
	t      TestingT
	name   string
	cancel context.CancelCauseFunc
	done   chan struct{}
	err    error
}

// Stop cancels the context passed to the function.
func (t *Task) Stop() {
	t.t.Helper()
	t.t.Logf("stopping %q", t.name)
	t.cancel(errStopped)
}

// StopAndWait cancels the context passed to the function and waits for it to
// return.
//
// If it returns an error, the test fails.
func (t *Task) StopAndWait() {
	t.t.Helper()
	t.t.Logf("stopping %q and waiting for it to return", t.name)
	t.cancel(errStopped)

	select {
	case <-time.After(shutdownTimeout):
		t.t.Errorf("%q did not return within %s of being stopped", t.name, shutdownTimeout)
	case <-t.done:
		if t.err != nil && t.err != errStopped {
			t.t.Fatalf("%q returned an error, expected it to be stopped explicitly: %s", t.name, t.err)
		}
	}
}

// WaitForSuccess waits for the function to return successfully.
//
// If it returns an error, the test fails.
func (t *Task) WaitForSuccess() {
	t.t.Helper()
	t.t.Logf("waiting for %q to return", t.name)

	select {
	case <-time.After(shutdownTimeout):
		t.t.Errorf("%q did not return within %s", t.name, shutdownTimeout)
	case <-t.done:
		if t.err != nil {
			t.t.Fatalf("%q returned an error, expected it to return successfully: %s", t.name, t.err)
		}
	}
}

// WaitUntilStopped waits for the function to be explicitly stopped
//
// If it returns an error, the test fails.
func (t *Task) WaitUntilStopped() {
	t.t.Helper()
	t.t.Logf("waiting for %q to return", t.name)

	select {
	case <-time.After(shutdownTimeout):
		t.t.Errorf("%q did not return within %s", t.name, shutdownTimeout)
	case <-t.done:
		if t.err != nil && t.err != errStopped {
			t.t.Fatalf("%q returned an error, expected it to be stopped explicitly: %s", t.name, t.err)
		}
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
		panic("cannot use Err() before Done() is closed")
	}

	return t.err
}
