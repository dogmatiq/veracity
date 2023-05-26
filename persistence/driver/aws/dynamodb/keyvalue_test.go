package dynamodb_test

import (
	"context"
	"testing"
	"time"

	. "github.com/dogmatiq/veracity/persistence/driver/aws/dynamodb"
	"github.com/dogmatiq/veracity/persistence/kv"
)

func TestKeyValueStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	client := newClient(t)
	table := "kvstore"

	if err := CreateKeyValueStoreTable(ctx, client, table); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := deleteTable(ctx, client, table); err != nil {
			t.Fatal(err)
		}

		cancel()
	})

	kv.RunTests(
		t,
		func(t *testing.T) kv.Store {
			return &KeyValueStore{
				Client: client,
				Table:  table,
			}
		},
	)
}
