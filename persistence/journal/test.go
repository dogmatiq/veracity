package journal

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// RunTests runs tests that confirm a journal implementation behaves correctly.
func RunTests(
	t *testing.T,
	newStore func(t *testing.T) Store,
) {
	t.Run("type Store", func(t *testing.T) {
		t.Run("func Open()", func(t *testing.T) {
			t.Run("does not perform naive path concatenation", func(t *testing.T) {
				store := newStore(t)

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				paths := [][]string{
					{"foobar"},
					{"foo", "bar"},
					{"foob", "ar"},
					{"foo/bar"},
					{"foo/", "bar"},
					{"foo", "/bar"},
				}

				for i, path := range paths {
					j, err := store.Open(ctx, path...)
					if err != nil {
						t.Fatal(err)
					}
					defer j.Close()

					expect := []byte(fmt.Sprintf("<record-%d>", i))
					ok, err := j.Append(ctx, 0, expect)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
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
				}
			})

			t.Run("allows journals to be opened multiple times", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				store := newStore(t)

				j1, err := store.Open(ctx, "<journal>")
				if err != nil {
					t.Fatal(err)
				}
				defer j1.Close()

				j2, err := store.Open(ctx, "<journal>")
				if err != nil {
					t.Fatal(err)
				}
				defer j2.Close()

				expect := []byte("<record>")
				ok, err := j1.Append(ctx, 0, expect)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				actual, ok, err := j2.Read(ctx, 0)
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
	})

	t.Run("type Journal", func(t *testing.T) {
		t.Run("func Read()", func(t *testing.T) {
			t.Run("it returns false if the version doesn't exist", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

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

				ctx, j := setup(t, newStore)

				var expect [][]byte

				// Ensure we test with a version that becomes 2 digits long.
				for i := 0; i < 15; i++ {
					expect = append(
						expect,
						[]byte(fmt.Sprintf("<record-%d>", i)),
					)
				}

				for ver, rec := range expect {
					ok, err := j.Append(ctx, uint64(ver), rec)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}
				}

				for ver, rec := range expect {
					actual, ok, err := j.Read(ctx, uint64(ver))
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("expected record to exist")
					}

					if !bytes.Equal(rec, actual) {
						t.Fatalf(
							"unexpected record, want %q, got %q",
							string(rec),
							string(actual),
						)
					}
				}
			})
		})

		t.Run("func GetOldest()", func(t *testing.T) {
			t.Run("it returns the record that produced version 0 if there has been no truncation", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				expect := []byte("<record>")
				ok, err := j.Append(ctx, 0, expect)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				ver, actual, ok, err := j.GetOldest(ctx)
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

				if ver != 0 {
					t.Fatalf("unexpected version, want 0, got %d", ver)
				}
			})

			t.Run("it returns false if the journal is empty", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				_, _, ok, err := j.GetOldest(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatal("returned ok == true for non-existent record")
				}
			})

			t.Run("it returns the first non-truncated record", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				records := [][]byte{
					[]byte("<record-1>"),
					[]byte("<record-2>"),
					[]byte("<record-3>"),
					[]byte("<record-4>"),
					[]byte("<record-5>"),
				}

				for ver, rec := range records {
					ok, err := j.Append(ctx, uint64(ver), rec)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}
				}

				retainVersion := uint64(len(records) - 1)
				err := j.Truncate(ctx, retainVersion)
				if err != nil {
					t.Fatal(err)
				}

				expect := records[retainVersion]
				ver, actual, ok, err := j.GetOldest(ctx)
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

				if ver != retainVersion {
					t.Fatalf("unexpected version, want %d, got %d", retainVersion, ver)
				}
			})
		})

		t.Run("func Append()", func(t *testing.T) {
			t.Run("it returns true if the version doesn't exist", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Append(ctx, 0, []byte("<record>"))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}
			})

			t.Run("it returns false if the version already exists", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Append(ctx, 0, []byte("<prior>"))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				expect := []byte("<original>")
				ok, err = j.Append(ctx, 1, expect)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				ok, err = j.Append(ctx, 1, []byte("<modified>"))
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatal("expected an optimistic concurrency conflict")
				}

				actual, ok, err := j.Read(ctx, 1)
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
	})
}

func setup(
	t *testing.T,
	newStore func(t *testing.T) Store,
) (context.Context, Journal) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	store := newStore(t)

	j, err := store.Open(ctx, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := j.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return ctx, j
}
