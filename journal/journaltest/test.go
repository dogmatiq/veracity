package journaltest

import (
	"context"
	"testing"
	"time"

	"github.com/dogmatiq/veracity/journal"
)

type TestContext struct {
	Journal journal.Journal[[]byte]
	Cleanup func() error
}

func RunTests(
	t *testing.T,
	setup func() TestContext,
) {
	t.Run("func Read()", func(t *testing.T) {
		t.Run("it returns false if the version doesn't exist", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			tc := setup()
			defer func() {
				if err := tc.Cleanup(); err != nil {
					t.Fatal(err)
				}
			}()

			_, ok, err := tc.Journal.Read(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}

			if ok {
				t.Fatal("returned ok == true for non-existent record")
			}
		})
	})

	t.Run("func Write()", func(t *testing.T) {
		t.Run("it returns true if the version doesn't exist", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			tc := setup()
			defer func() {
				if err := tc.Cleanup(); err != nil {
					t.Fatal(err)
				}
			}()

			ok, err := tc.Journal.Write(ctx, 0, []byte("<record>"))
			if err != nil {
				t.Fatal(err)
			}

			if !ok {
				t.Fatal("unexpected optimistic concurrency conflict")
			}
		})
	})
}
