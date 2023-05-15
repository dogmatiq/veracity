package postgres_test

import (
	"context"
	"testing"

	"github.com/dogmatiq/sqltest"
	. "github.com/dogmatiq/veracity/persistence/driver/postgres"
	"github.com/dogmatiq/veracity/persistence/journal"
)

func TestJournal(t *testing.T) {
	ctx := context.Background()
	database, err := sqltest.NewDatabase(ctx, sqltest.PGXDriver, sqltest.PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}

	db, err := database.Open()
	if err != nil {
		t.Fatal(err)
	}

	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}

		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
	})

	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{
				DB: db,
			}
		},
	)
}
