package journal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dogmatiq/veracity/internal/test"
)

// RunTests runs tests that confirm a journal implementation behaves correctly.
func RunTests(
	t *testing.T,
	newStore func(t *testing.T) Store,
) {
	type dependencies struct {
		Store       Store
		JournalName string
		Journal     Journal
	}

	setup := func(
		t *testing.T,
		newStore func(t *testing.T) Store,
	) *dependencies {
		deps := &dependencies{
			Store:       newStore(t),
			JournalName: fmt.Sprintf("<journal-%d>", journalCounter.Add(1)),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		j, err := deps.Store.Open(ctx, deps.JournalName)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			if err := j.Close(); err != nil {
				t.Fatal(err)
			}
		})

		deps.Journal = j

		return deps
	}

	t.Run("type Store", func(t *testing.T) {
		t.Parallel()

		t.Run("func Open()", func(t *testing.T) {
			t.Parallel()

			t.Run("allows a journal to be opened multiple times", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				store := newStore(t)

				j1, err := store.Open(tctx, "<journal>")
				if err != nil {
					t.Fatal(err)
				}
				defer j1.Close()

				j2, err := store.Open(tctx, "<journal>")
				if err != nil {
					t.Fatal(err)
				}
				defer j2.Close()

				want := []byte("<record>")
				if err := j1.Append(tctx, 0, want); err != nil {
					t.Fatal(err)
				}

				got, err := j2.Get(tctx, 0)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected journal record",
					got,
					want,
				)
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
					ExpectBegin, ExpectEnd Position
					Setup                  func(context.Context, *testing.T, Journal)
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

						tctx := test.WithContext(t)
						deps := setup(t, newStore)

						c.Setup(tctx, t, deps.Journal)

						begin, end, err := deps.Journal.Bounds(tctx)
						if err != nil {
							t.Fatal(err)
						}

						test.Expect(
							t,
							"unexpected begin position",
							begin,
							c.ExpectBegin,
						)

						test.Expect(
							t,
							"unexpected end position",
							end,
							c.ExpectEnd,
						)
					})
				}
			})
		})

		t.Run("func Get()", func(t *testing.T) {
			t.Parallel()

			t.Run("it returns ErrNotFound if there is no record at the given position", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				_, err := deps.Journal.Get(tctx, 1)
				if !errors.Is(err, ErrNotFound) {
					t.Fatal(err)
				}
			})

			t.Run("it returns the record if it exists", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				// Ensure we test with a position that becomes 2 digits long to
				// confirm that the implementation is not using a lexical sort.
				records := appendRecords(tctx, t, deps.Journal, 15)

				for i, want := range records {
					got, err := deps.Journal.Get(tctx, Position(i))
					if err != nil {
						t.Fatal(err)
					}

					test.Expect(
						t,
						fmt.Sprintf("unexpected record at position %d", i),
						got,
						want,
					)

					if !bytes.Equal(want, got) {
						t.Fatalf(
							"unexpected record, want %q, got %q",
							string(want),
							string(got),
						)
					}
				}
			})

			t.Run("it does not return its internal byte slice", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 1)

				rec, err := deps.Journal.Get(tctx, 0)
				if err != nil {
					t.Fatal(err)
				}

				rec[0] = 'X'

				got, err := deps.Journal.Get(tctx, 0)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected record",
					got,
					[]byte("<record-0>"),
				)
			})
		})

		t.Run("func Range()", func(t *testing.T) {
			t.Parallel()

			t.Run("calls the function for each record in the journal", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				want := appendRecords(tctx, t, deps.Journal, 15)

				var got [][]byte
				wantPos := Position(10)
				want = want[wantPos:]

				if err := deps.Journal.Range(
					tctx,
					wantPos,
					func(ctx context.Context, gotPos Position, rec []byte) (bool, error) {
						test.Expect(
							t,
							"unexpected position",
							gotPos,
							wantPos,
						)

						got = append(got, rec)
						wantPos++

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected records",
					got,
					want,
				)
			})

			t.Run("it stops iterating if the function returns false", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 2)

				called := false
				if err := deps.Journal.Range(
					tctx,
					0,
					func(ctx context.Context, pos Position, rec []byte) (bool, error) {
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

			t.Run("it returns ErrNotFound if the first record is truncated", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				records := appendRecords(tctx, t, deps.Journal, 5)
				retainPos := Position(len(records) - 1)

				err := deps.Journal.Truncate(tctx, retainPos)
				if err != nil {
					t.Fatal(err)
				}

				err = deps.Journal.Range(
					tctx,
					1,
					func(ctx context.Context, pos Position, rec []byte) (bool, error) {
						panic("unexpected call")
					},
				)

				test.Expect(
					t,
					"unexpected error",
					err,
					ErrNotFound,
				)
			})

			t.Run("it returns an error if a record is truncated during iteration", func(t *testing.T) {
				t.Skip() // TODO
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 5)

				err := deps.Journal.Range(
					tctx,
					0,
					func(ctx context.Context, pos Position, rec []byte) (bool, error) {
						return true, deps.Journal.Truncate(ctx, 5)
					},
				)

				test.Expect(
					t,
					"unexpected error",
					err,
					ErrNotFound,
				)
			})

			t.Run("it does not invoke the function with its internal byte slice", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 1)

				if err := deps.Journal.Range(
					tctx,
					0,
					func(ctx context.Context, pos Position, rec []byte) (bool, error) {
						rec[0] = 'X'

						return true, nil
					},
				); err != nil {
					t.Fatal(err)
				}

				got, err := deps.Journal.Get(tctx, 0)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected record",
					got,
					[]byte("<record-0>"),
				)
			})
		})

		t.Run("func Append()", func(t *testing.T) {
			t.Parallel()

			t.Run("it does not return an error if there is no record at the given position", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				if err := deps.Journal.Append(tctx, 0, []byte("<record>")); err != nil {
					t.Fatal(err)
				}
			})

			t.Run("it returns ErrConflict there is already a record at the given position", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				if err := deps.Journal.Append(tctx, 0, []byte("<prior>")); err != nil {
					t.Fatal(err)
				}

				want := []byte("<original>")
				if err := deps.Journal.Append(tctx, 1, want); err != nil {
					t.Fatal(err)
				}

				err := deps.Journal.Append(tctx, 1, []byte("<modified>"))

				test.Expect(
					t,
					"unexpected error",
					err,
					ErrConflict,
				)

				got, err := deps.Journal.Get(tctx, 1)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected record",
					got,
					want,
				)
			})

			t.Run("it does not keep a reference to the record slice", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				rec := []byte("<record>")

				if err := deps.Journal.Append(tctx, 0, rec); err != nil {
					t.Fatal(err)
				}

				rec[0] = 'X'

				got, err := deps.Journal.Get(tctx, 0)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected record",
					got,
					[]byte("<record>"),
				)
			})
		})

		t.Run("func Truncate()", func(t *testing.T) {
			t.Parallel()

			t.Run("it truncates the journal", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 3)

				if err := deps.Journal.Truncate(tctx, 1); err != nil {
					t.Fatal(err)
				}

				begin, _, err := deps.Journal.Bounds(tctx)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected begin position",
					begin,
					1,
				)
			})

			t.Run("it truncates the journal when it has already been truncated", func(t *testing.T) {
				t.Parallel()

				tctx := test.WithContext(t)
				deps := setup(t, newStore)

				appendRecords(tctx, t, deps.Journal, 3)

				if err := deps.Journal.Truncate(tctx, 1); err != nil {
					t.Fatal(err)
				}

				if err := deps.Journal.Truncate(tctx, 2); err != nil {
					t.Fatal(err)
				}

				begin, _, err := deps.Journal.Bounds(tctx)
				if err != nil {
					t.Fatal(err)
				}

				test.Expect(
					t,
					"unexpected begin position",
					begin,
					2,
				)
			})
		})
	})
}

var journalCounter atomic.Uint64

// appendRecords appends records to j.
func appendRecords(
	ctx context.Context,
	t test.FailerT,
	j Journal,
	n int,
) [][]byte {
	var records [][]byte

	for pos := Position(0); pos < Position(n); pos++ {
		rec := []byte(
			fmt.Sprintf("<record-%d>", pos),
		)

		records = append(records, rec)

		if err := j.Append(ctx, pos, rec); err != nil {
			t.Fatal(err)
		}
	}

	return records
}
