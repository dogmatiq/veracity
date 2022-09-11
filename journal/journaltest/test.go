package journaltest

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dogmatiq/veracity/journal"
)

// RunTests runs tests that confirm a journal implementation behaves correctly.
func RunTests(
	t *testing.T,
	new func(t *testing.T) journal.BinaryJournal,
) {
	t.Run("func Read()", func(t *testing.T) {
		t.Run("it returns false if the version doesn't exist", func(t *testing.T) {
			t.Parallel()

			ctx, j := setup(t, new)

			_, ok, err := j.Read(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("returned ok == true for non-existent record")
			}
		})

		t.Run("it returns the record if it exists", func(t *testing.T) {
			t.Parallel()

			ctx, j := setup(t, new)

			expect := [][]byte{
				[]byte("<record-1>"),
				[]byte("<record-2>"),
			}

			for i, r := range expect {
				ok, err := j.Write(ctx, uint64(i), r)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}
			}

			for i, r := range expect {
				actual, ok, err := j.Read(ctx, uint64(i))
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

			ctx, j := setup(t, new)

			ok, err := j.Write(ctx, 0, []byte("<record>"))
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("unexpected optimistic concurrency conflict")
			}
		})

		t.Run("it returns false if the version already exists", func(t *testing.T) {
			t.Parallel()

			ctx, j := setup(t, new)

			expect := []byte("<original>")

			ok, err := j.Write(ctx, 0, expect)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("unexpected optimistic concurrency conflict")
			}

			ok, err = j.Write(ctx, 0, []byte("<modified>"))
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("expected an optimistic concurrency conflict")
			}

			actual, ok, err := j.Read(ctx, 0)
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

func setup(
	t *testing.T,
	new func(t *testing.T) journal.BinaryJournal,
) (context.Context, journal.BinaryJournal) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	j := new(t)

	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return ctx, j
}
