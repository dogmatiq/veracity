package journal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
				if err := j1.Append(ctx, 0, expect); err != nil {
					t.Fatal(err)
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

		t.Run("func Bounds()", func(t *testing.T) {
			t.Parallel()

			t.Run("it returns the expected bounds", func(t *testing.T) {
				cases := []struct {
					Desc                   string
					ExpectBegin, ExpectEnd uint64
					Setup                  func(ctx context.Context, t *testing.T, j Journal)
				}{
					{
						"empty",
						0, 0,
						func(ctx context.Context, t *testing.T, j Journal) {},
					},
					{
						"with records",
						0, 10,
						func(ctx context.Context, t *testing.T, j Journal) {
							appendRecords(ctx, t, j, 10)
						},
					},
					{
						"with truncated records",
						5, 10,
						func(ctx context.Context, t *testing.T, j Journal) {
							appendRecords(ctx, t, j, 10)
							if err := j.Truncate(ctx, 5); err != nil {
								t.Fatal(err)
							}
						},
					},
				}

				for _, c := range cases {
					c := c // capture loop variable
					t.Run(c.Desc, func(t *testing.T) {
						t.Parallel()

						ctx, j := setup(t, newStore)
						c.Setup(ctx, t, j)

						begin, end, err := j.Bounds(ctx)
						if err != nil {
							t.Fatal(err)
						}

						if begin != c.ExpectBegin {
							t.Fatalf("unexpected begin offset, want %d, got %d", c.ExpectBegin, begin)
						}

						if end != c.ExpectEnd {
							t.Fatalf("unexpected end offset, want %d, got %d", c.ExpectEnd, end)
						}
					})
				}
			})
		})

		t.Run("func Get()", func(t *testing.T) {
			t.Parallel()

			t.Run("it returns false if there is no record at the given offset", func(t *testing.T) {
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

				// Ensure we test with an offset that becomes 2 digits long to
				// confirm that the implementation is not using a lexical sort
				// on the offset.
				expect := appendRecords(ctx, t, j, 15)

				for offset, rec := range expect {
					actual, ok, err := j.Get(ctx, uint64(offset))
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
				appendRecords(ctx, t, j, 1)

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

				if expect := []byte("<record-0>"); !bytes.Equal(expect, actual) {
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
				expect := appendRecords(ctx, t, j, 15)

				var actual [][]byte
				expectOffset := uint64(10)
				expect = expect[expectOffset:]

				if err := j.Range(
					ctx,
					expectOffset,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
						if offset != expectOffset {
							t.Fatalf("unexpected offset: want %d, got %d", expectOffset, offset)
						}

						actual = append(actual, rec)
						expectOffset++

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
				appendRecords(ctx, t, j, 2)

				called := false
				if err := j.Range(
					ctx,
					0,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
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
				records := appendRecords(ctx, t, j, 5)
				retainOffset := uint64(len(records) - 1)

				err := j.Truncate(ctx, retainOffset)
				if err != nil {
					t.Fatal(err)
				}

				err = j.Range(
					ctx,
					1,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
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
				appendRecords(ctx, t, j, 1)

				if err := j.Range(
					ctx,
					0,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
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

				if expect := []byte("<record-0>"); !bytes.Equal(expect, actual) {
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
				expect := appendRecords(ctx, t, j, 15)

				var actual [][]byte
				var expectOffset uint64

				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
						if offset != expectOffset {
							t.Fatalf("unexpected offset: want %d, got %d", expectOffset, offset)
						}

						actual = append(actual, rec)
						expectOffset++

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
				appendRecords(ctx, t, j, 2)

				called := false
				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
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
				records := appendRecords(ctx, t, j, 5)

				retainOffset := uint64(len(records) - 1)
				err := j.Truncate(ctx, retainOffset)
				if err != nil {
					t.Fatal(err)
				}

				if err := j.RangeAll(
					ctx,
					func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
						if offset != retainOffset {
							t.Fatalf("unexpected offset: want %d, got %d", retainOffset, offset)
						}

						if !bytes.Equal(rec, records[retainOffset]) {
							t.Fatalf("unexpected record: want %q, got %q", records[retainOffset], rec)
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
				appendRecords(ctx, t, j, 1)

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

				if expect := []byte("<record-0>"); !bytes.Equal(expect, actual) {
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

			t.Run("it does not return an error if there is no record at the given offset", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				if err := j.Append(ctx, 0, []byte("<record>")); err != nil {
					t.Fatal(err)
				}
			})

			t.Run("it returns ErrConflict there is already a record at the given offset", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				if err := j.Append(ctx, 0, []byte("<prior>")); err != nil {
					t.Fatal(err)
				}

				expect := []byte("<original>")
				if err := j.Append(ctx, 1, expect); err != nil {
					t.Fatal(err)
				}

				err := j.Append(ctx, 1, []byte("<modified>"))
				if err == nil {
					t.Fatal("expected ErrConflict")
				}
				if err != ErrConflict {
					t.Fatal(err)
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

				if err := j.Append(ctx, 0, rec); err != nil {
					t.Fatal(err)
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

		t.Run("func Truncate()", func(t *testing.T) {
			t.Parallel()

			t.Run("it truncates the journal", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)
				appendRecords(ctx, t, j, 3)

				if err := j.Truncate(ctx, 1); err != nil {
					t.Fatal(err)
				}

				begin, _, err := j.Bounds(ctx)
				if err != nil {
					t.Fatal(err)
				}

				const expect = 1
				if begin != expect {
					t.Fatalf("unexpected begin offset, want %d, got %d", expect, begin)
				}
			})

			t.Run("it truncates the journal when it has already been truncated", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)
				appendRecords(ctx, t, j, 3)

				if err := j.Truncate(ctx, 1); err != nil {
					t.Fatal(err)
				}

				if err := j.Truncate(ctx, 2); err != nil {
					t.Fatal(err)
				}

				begin, _, err := j.Bounds(ctx)
				if err != nil {
					t.Fatal(err)
				}

				const expect = 2
				if begin != expect {
					t.Fatalf("unexpected begin offset, want %d, got %d", expect, begin)
				}
			})
		})
	})
}

var journalCounter atomic.Uint64

func setup(
	t *testing.T,
	newStore func(t *testing.T) Store,
) (context.Context, Journal) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	store := newStore(t)

	name := fmt.Sprintf("<journal-%d>", journalCounter.Add(1))
	j, err := store.Open(ctx, name)
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

// appendRecords appends records to j.
func appendRecords(
	ctx context.Context,
	t interface{ Fatal(...interface{}) },
	j Journal,
	n uint64,
) [][]byte {
	var records [][]byte

	for offset := uint64(0); offset < n; offset++ {
		rec := []byte(
			fmt.Sprintf("<record-%d>", offset),
		)

		records = append(records, rec)

		if err := j.Append(ctx, offset, rec); err != nil {
			t.Fatal(err)
		}
	}

	return records
}
