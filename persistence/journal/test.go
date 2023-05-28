package journal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

// RunTests runs tests that confirm a journal implementation behaves correctly.
func RunTests(
	t *testing.T,
	newStore func(t *testing.T) Store,
) {
	t.Run("type Store", func(t *testing.T) {
		t.Parallel()

		t.Run("func Open()", func(t *testing.T) {
			t.Parallel()

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

				actual, ok, err := j2.Get(ctx, 0)
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
		t.Parallel()

		t.Run("func Get()", func(t *testing.T) {
			t.Parallel()

			t.Run("it returns false if the version doesn't exist", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				_, ok, err := j.Get(ctx, 1)
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
					actual, ok, err := j.Get(ctx, uint64(ver))
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

			t.Run("it does not return its internal byte slice", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Append(ctx, 0, []byte("<record>"))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				rec, ok, err := j.Get(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				rec[0] = 'X'

				actual, ok, err := j.Get(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				if expect := []byte("<record>"); !bytes.Equal(expect, actual) {
					t.Fatalf(
						"unexpected record, want %q, got %q",
						string(expect),
						string(actual),
					)
				}
			})
		})

		t.Run("func Range()", func(t *testing.T) {
			t.Parallel()

			t.Run("calls the function for each record in the journal", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				var expect [][]byte

				for ver := uint64(0); ver < 100; ver++ {
					rec := []byte(fmt.Sprintf("<record-%d>", ver))
					ok, err := j.Append(ctx, ver, rec)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}

					expect = append(expect, rec)
				}

				var actual [][]byte
				expectVer := uint64(50)
				expect = expect[expectVer:]

				if err := j.Range(
					ctx,
					expectVer,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						if ver != expectVer {
							t.Fatalf("unexpected version: want %d, got %d", expectVer, ver)
						}

						actual = append(actual, rec)
						expectVer++

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(expect, actual); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("it stops iterating if the function returns false", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				for ver := uint64(0); ver < 2; ver++ {
					ok, err := j.Append(ctx, ver, []byte("<record>"))
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}
				}

				called := false
				if err := j.Range(
					ctx,
					0,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						if called {
							return false, errors.New("unexpected call")
						}

						called = true
						return false, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			})

			t.Run("returns an error if the first record is truncated", func(t *testing.T) {
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

				err = j.Range(
					ctx,
					1,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						panic("unexpected call")
					},
				)
				if err == nil {
					t.Fatal("expected error")
				}

				expect := "cannot range over truncated records"
				if err.Error() != expect {
					t.Fatalf("unexpected error: want %s, got %s", expect, err.Error())
				}
			})

			t.Run("it does not invoke the function with its internal byte slice", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Append(ctx, 0, []byte("<record>"))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				if err := j.Range(
					ctx,
					0,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						rec[0] = 'X'

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				actual, ok, err := j.Get(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				if expect := []byte("<record>"); !bytes.Equal(expect, actual) {
					t.Fatalf(
						"unexpected record, want %q, got %q",
						string(expect),
						string(actual),
					)
				}
			})
		})

		t.Run("func RangeAll()", func(t *testing.T) {
			t.Parallel()

			t.Run("calls the function for each record in the journal", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				var expect [][]byte

				for ver := uint64(0); ver < 100; ver++ {
					rec := []byte(fmt.Sprintf("<record-%d>", ver))
					ok, err := j.Append(ctx, ver, rec)
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}

					expect = append(expect, rec)
				}

				var actual [][]byte
				var expectVer uint64

				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						if ver != expectVer {
							t.Fatalf("unexpected version: want %d, got %d", expectVer, ver)
						}

						actual = append(actual, rec)
						expectVer++

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(expect, actual); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("it stops iterating if the function returns false", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				for ver := uint64(0); ver < 2; ver++ {
					ok, err := j.Append(ctx, ver, []byte("<record>"))
					if err != nil {
						t.Fatal(err)
					}
					if !ok {
						t.Fatal("unexpected optimistic concurrency conflict")
					}
				}

				called := false
				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						if called {
							return false, errors.New("unexpected call")
						}

						called = true
						return false, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			})

			t.Run("it starts at the first non-truncated record", func(t *testing.T) {
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

				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
						if ver != retainVersion {
							t.Fatalf("unexpected version: want %d, got %d", retainVersion, ver)
						}

						if !bytes.Equal(rec, records[retainVersion]) {
							t.Fatalf("unexpected record: want %q, got %q", records[retainVersion], rec)
						}

						return false, nil
					},
				); err != nil {
					t.Fatal(err)
				}
			})

			t.Run("it does not invoke the function with its internal byte slice", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Append(ctx, 0, []byte("<record>"))
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, _ uint64, rec []byte) (bool, error) {
						rec[0] = 'X'

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				actual, ok, err := j.Get(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				if expect := []byte("<record>"); !bytes.Equal(expect, actual) {
					t.Fatalf(
						"unexpected record, want %q, got %q",
						string(expect),
						string(actual),
					)
				}
			})
		})

		t.Run("func Append()", func(t *testing.T) {
			t.Parallel()

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

				actual, ok, err := j.Get(ctx, 1)
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

			t.Run("it does not keep a reference to the record slice", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				rec := []byte("<record>")

				ok, err := j.Append(ctx, 0, rec)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("unexpected optimistic concurrency conflict")
				}

				rec[0] = 'X'

				actual, ok, err := j.Get(ctx, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected record to exist")
				}

				if expect := []byte("<record>"); !bytes.Equal(expect, actual) {
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
