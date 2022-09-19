package kv

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
					func() {
						ks, err := store.Open(ctx, path...)
						if err != nil {
							t.Fatal(err)
						}
						defer ks.Close()

						expect := []byte(fmt.Sprintf("<value-%d>", i))
						if err := ks.Set(ctx, []byte("<key>"), expect); err != nil {
							t.Fatal(err)
						}
					}()
				}

				for i, path := range paths {
					func() {
						ks, err := store.Open(ctx, path...)
						if err != nil {
							t.Fatal(err)
						}
						defer ks.Close()

						expect := []byte(fmt.Sprintf("<value-%d>", i))
						actual, err := ks.Get(ctx, []byte("<key>"))
						if err != nil {
							t.Fatal(err)
						}

						if !bytes.Equal(expect, actual) {
							t.Fatalf(
								"unexpected record, want %q, got %q",
								string(expect),
								string(actual),
							)
						}
					}()
				}
			})

			t.Run("allows keyspaces to be opened multiple times", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				store := newStore(t)

				ks1, err := store.Open(ctx, "<keyspace>")
				if err != nil {
					t.Fatal(err)
				}
				defer ks1.Close()

				ks2, err := store.Open(ctx, "<keyspace>")
				if err != nil {
					t.Fatal(err)
				}
				defer ks2.Close()

				expect := []byte("<value>")
				if err := ks1.Set(ctx, []byte("<key>"), expect); err != nil {
					t.Fatal(err)
				}

				actual, err := ks2.Get(ctx, []byte("<key>"))
				if err != nil {
					t.Fatal(err)
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

	t.Run("type Keyspace", func(t *testing.T) {
		t.Run("func Get()", func(t *testing.T) {
			t.Run("it returns an empty value if the key doesn't exist", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				v, err := j.Get(ctx, []byte("<key>"))
				if err != nil {
					t.Fatal(err)
				}
				if len(v) != 0 {
					t.Fatal("expected zero-length value")
				}
			})

			t.Run("it returns an empty value if the key has been deleted", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				k := []byte("<key>")

				if err := j.Set(ctx, k, []byte("<value>")); err != nil {
					t.Fatal(err)
				}

				if err := j.Set(ctx, k, nil); err != nil {
					t.Fatal(err)
				}

				v, err := j.Get(ctx, k)
				if err != nil {
					t.Fatal(err)
				}
				if len(v) != 0 {
					t.Fatal("expected zero-length value")
				}
			})

			t.Run("it returns the value if the key exists", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				for i := 0; i < 5; i++ {
					k := []byte(fmt.Sprintf("<key-%d>", i))
					v := []byte(fmt.Sprintf("<value-%d>", i))

					if err := j.Set(ctx, k, v); err != nil {
						t.Fatal(err)
					}
				}

				for i := 0; i < 5; i++ {
					k := []byte(fmt.Sprintf("<key-%d>", i))
					expect := []byte(fmt.Sprintf("<value-%d>", i))

					actual, err := j.Get(ctx, k)
					if err != nil {
						t.Fatal(err)
					}

					if !bytes.Equal(expect, actual) {
						t.Fatalf(
							"unexpected value, want %q, got %q",
							string(expect),
							string(actual),
						)
					}
				}
			})
		})

		t.Run("func Has()", func(t *testing.T) {
			t.Run("it returns false if the key doesn't exist", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				ok, err := j.Has(ctx, []byte("<key>"))
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatal("expected ok to be false")
				}
			})

			t.Run("it returns true if the key exists", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				k := []byte("<key>")

				if err := j.Set(ctx, k, []byte("<value>")); err != nil {
					t.Fatal(err)
				}

				ok, err := j.Has(ctx, k)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatal("expected ok to be true")
				}
			})

			t.Run("it returns false if the key has been deleted", func(t *testing.T) {
				t.Parallel()

				ctx, j := setup(t, newStore)

				k := []byte("<key>")

				if err := j.Set(ctx, k, []byte("<value>")); err != nil {
					t.Fatal(err)
				}

				if err := j.Set(ctx, k, nil); err != nil {
					t.Fatal(err)
				}

				ok, err := j.Has(ctx, k)
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatal("expected ok to be false")
				}
			})
		})
	})
}

func setup(
	t *testing.T,
	newStore func(t *testing.T) Store,
) (context.Context, Keyspace) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	store := newStore(t)

	ks, err := store.Open(ctx, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := ks.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return ctx, ks
}
