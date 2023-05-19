package postgres_test

import (
	"context"
	"testing"

	"github.com/dogmatiq/sqltest"
	. "github.com/dogmatiq/veracity/persistence/driver/postgres"
	"github.com/dogmatiq/veracity/persistence/kv"
)

func TestKV(t *testing.T) {
	ctx := context.Background()

	database, err := sqltest.NewDatabase(ctx, sqltest.PGXDriver, sqltest.PostgreSQL)
	if err != nil {
		t.Fatal(err)
	}

	db, err := database.Open()
	if err != nil {
		t.Fatal(err)
	}

	if err := CreateKVStoreSchema(ctx, db); err != nil {
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

	kv.RunTests(
		t,
		func(t *testing.T) kv.Store {
			return &KeyValueStore{
				DB: db,
			}
		},
	)
}
