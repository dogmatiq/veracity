package future_test

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/dogmatiq/veracity/internal/future"
)

func TestFailable(t *testing.T) {
	t.Parallel()

	t.Run("func Ready()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns a channel that is closed when the future is resolved with a value", func(t *testing.T) {
			t.Parallel()

			future, resolver := NewFailable[int]()

			go func() {
				time.Sleep(50 * time.Millisecond)
				resolver.Set(42)
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

			future, resolver := NewFailable[int]()

			go func() {
				time.Sleep(50 * time.Millisecond)
				resolver.Err(errors.New("<error>"))
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

			future, resolver := NewFailable[int]()

			expect := 42
			resolver.Set(expect)

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
				expect := "future value is not ready"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			future, _ := NewFailable[int]()
			future.Get()
		})
	})

	t.Run("func Wait()", func(t *testing.T) {
		t.Parallel()

		t.Run("it blocks until the future is resolved with a value", func(t *testing.T) {
			t.Parallel()

			future, resolver := NewFailable[int]()

			expect := 42

			go func() {
				time.Sleep(50 * time.Millisecond)
				resolver.Set(expect)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			actual, err := future.Wait(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}

			// Additional calls must return the same value.
			actual, err = future.Wait(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if actual != expect {
				t.Fatalf("unexpected value: got %d, want %d", actual, expect)
			}
		})

		t.Run("it blocks until the future is resolved with an error", func(t *testing.T) {
			t.Parallel()

			future, resolver := NewFailable[int]()

			expect := errors.New("<error>")

			go func() {
				time.Sleep(50 * time.Millisecond)
				resolver.Err(expect)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if _, err := future.Wait(ctx); err != expect {
				t.Fatalf("unexpected value: got %q, want %q", err, expect)
			}

			// Additional calls must return the same error.
			if _, err := future.Wait(ctx); err != expect {
				t.Fatalf("unexpected value: got %q, want %q", err, expect)
			}
		})

		t.Run("it returns an error if the context is cancelled", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			future, _ := NewFailable[int]()

			_, err := future.Wait(ctx)
			if err != context.DeadlineExceeded {
				t.Fatal(err)
			}
		})
	})

	t.Run("func Set()", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics if the future is already resolved", func(t *testing.T) {
			t.Parallel()

			_, resolver := NewFailable[int]()
			resolver.Set(42)

			defer func() {
				expect := "future has already been resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			resolver.Set(42)
		})
	})

	t.Run("func Err()", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics if the future is already resolved", func(t *testing.T) {
			t.Parallel()

			_, resolver := NewFailable[int]()
			resolver.Set(42)

			defer func() {
				expect := "future has already been resolved"
				if actual := recover(); actual != expect {
					t.Fatalf("unexpected panic value: got %v, want %q", actual, expect)
				}
			}()

			resolver.Err(errors.New("<error>"))
		})
	})

	t.Run("func IsResolved()", func(t *testing.T) {
		t.Parallel()

		t.Run("it returns true if the future is resolved with a value", func(t *testing.T) {
			t.Parallel()

			_, resolver := NewFailable[int]()

			if resolver.IsResolved() {
				t.Fatal("did not expect future to be resolved")
			}

			resolver.Set(42)

			if !resolver.IsResolved() {
				t.Fatal("expected future to be resolved")
			}
		})

		t.Run("it returns true if the future is resolved with an error", func(t *testing.T) {
			t.Parallel()

			_, resolver := NewFailable[int]()

			if resolver.IsResolved() {
				t.Fatal("did not expect future to be resolved")
			}

			resolver.Err(errors.New("<error>"))

			if !resolver.IsResolved() {
				t.Fatal("expected future to be resolved")
			}
		})
	})
}
