package journaltest

import (
	"bytes"
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
	setup func() (TestContext, error),
) {
	t.Run("func Read()", func(t *testing.T) {
		t.Run("it returns false if the version doesn't exist", func(t *testing.T) {
			t.Parallel()

			ctx, tc := prepare(t, setup)

			_, ok, err := tc.Journal.Read(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("returned ok == true for non-existent record")
			}
		})

		t.Run("it returns the record if it exists", func(t *testing.T) {
			t.Parallel()

			ctx, tc := prepare(t, setup)

			expect := [][]byte{
				[]byte("<record-1>"),
				[]byte("<record-2>"),
			}

			for i, r := range expect {
				ok, err := tc.Journal.Write(ctx, uint64(i), r)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}
			}

			for i, r := range expect {
				actual, ok, err := tc.Journal.Read(ctx, uint64(i))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				if !bytes.Equal(r, actual) {
					t.Fatalf(
						"unexpected record, want %q, got %q",
						string(r),
						string(actual),
					)
				}
			}
		})
	})

	t.Run("func Write()", func(t *testing.T) {
		t.Run("it returns true if the version doesn't exist", func(t *testing.T) {
			t.Parallel()

			ctx, tc := prepare(t, setup)

			ok, err := tc.Journal.Write(ctx, 0, []byte("<record>"))
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("unexpected optimistic concurrency conflict")
			}
		})

		t.Run("it returns false if the version already exists", func(t *testing.T) {
			t.Parallel()

			ctx, tc := prepare(t, setup)

			expect := []byte("<original>")

			ok, err := tc.Journal.Write(ctx, 0, expect)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("unexpected optimistic concurrency conflict")
			}

			ok, err = tc.Journal.Write(ctx, 0, []byte("<modified>"))
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("expected an optimistic concurrency conflict")
			}

			actual, ok, err := tc.Journal.Read(ctx, 0)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("expected record to exist")
			}

			if !bytes.Equal(expect, actual) {
				t.Fatalf(
					"unexpected record, want %q, got %q",
					string(expect),
					string(actual),
				)
			}
		})
	})
}

func prepare(
	t *testing.T,
	setup func() (TestContext, error),
) (context.Context, TestContext) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	tc, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := tc.Cleanup(); err != nil {
			t.Fatal(err)
		}
	})

	return ctx, tc
}
