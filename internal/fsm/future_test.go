package fsm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/dogmatiq/veracity/internal/fsm"
)

func TestFuture(t *testing.T) {
	t.Parallel()

	t.Run("func Ready()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns a channel that is closed when the future is resolved", func(t *testing.T) {
			t.Parallel()

			var future Future[int]

			go func() {
				time.Sleep(50 * time.Millisecond)
				future.Set(42)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			select {
			case <-ctx.Done():
				t.Fatal("timed out waiting for future to be resolved")
			case <-future.Ready():
				// ok
			}
		})
	})
	t.Run("func Get()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns the value if the future is resolved", func(t *testing.T) {
			t.Parallel()

			var future Future[int]

			expect := 42
			future.Set(expect)

			if actual := future.Get(); actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}

			// Additional calls must return the same value.
			if actual := future.Get(); actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}
		})

		t.Run("it panics if the future is not resolved", func(t *testing.T) {
			t.Parallel()

			defer func() {
				expect := "future is not resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			var future Future[int]
			future.Get()
		})
	})

	t.Run("func Set()", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics if the future is already resolved", func(t *testing.T) {
			t.Parallel()

			var future Future[int]

			future.Set(42)

			defer func() {
				expect := "future is already resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			future.Set(42)
		})
	})
}

func TestFailableFuture(t *testing.T) {
	t.Parallel()

	t.Run("func Ready()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns a channel that is closed when the future is resolved with a value", func(t *testing.T) {
			t.Parallel()

			var future FailableFuture[int]

			go func() {
				time.Sleep(50 * time.Millisecond)
				future.Set(42)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			select {
			case <-ctx.Done():
				t.Fatal("timed out waiting for future to be resolved")
			case <-future.Ready():
				// ok
			}
		})

		t.Run("it returns a channel that is closed when the future is resolved with an error", func(t *testing.T) {
			t.Parallel()

			var future FailableFuture[int]

			go func() {
				time.Sleep(50 * time.Millisecond)
				future.Err(errors.New("<error>"))
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			select {
			case <-ctx.Done():
				t.Fatal("timed out waiting for future to be resolved")
			case <-future.Ready():
				// ok
			}
		})
	})

	t.Run("func Get()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns the value if the future is resolved with a value", func(t *testing.T) {
			t.Parallel()

			var future FailableFuture[int]

			expect := 42
			future.Set(expect)

			actual, err := future.Get()
			if err != nil {
				t.Fatal(err)
			}
			if actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}

			// Additional calls must return the same value.
			actual, err = future.Get()
			if err != nil {
				t.Fatal(err)
			}
			if actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}
		})

		t.Run("it panics if the future is not resolved", func(t *testing.T) {
			t.Parallel()

			defer func() {
				expect := "future is not resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			var future FailableFuture[int]
			future.Get()
		})
	})

	t.Run("func Set()", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics if the future is already resolved", func(t *testing.T) {
			t.Parallel()

			var future FailableFuture[int]

			future.Set(42)

			defer func() {
				expect := "future is already resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			future.Set(42)
		})
	})

	t.Run("func Err()", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics if the future is already resolved", func(t *testing.T) {
			t.Parallel()

			var future FailableFuture[int]

			future.Set(42)

			defer func() {
				expect := "future is already resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			future.Err(errors.New("<error>"))
		})
	})
}
