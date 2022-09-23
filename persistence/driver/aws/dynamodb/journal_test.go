package dynamodb_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	. "github.com/dogmatiq/veracity/persistence/driver/aws/dynamodb"
	"github.com/dogmatiq/veracity/persistence/driver/aws/internal/awsx"
	"github.com/dogmatiq/veracity/persistence/journal"
)

func TestJournal(t *testing.T) {
	endpoint := os.Getenv("DOGMATIQ_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:28000"
	}

	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials("<id>", "<secret>", ""),
		Endpoint:    aws.String(endpoint),
		Region:      aws.String("us-east-1"),
		DisableSSL:  aws.Bool(true),
	}

	sess, err := session.NewSession(config)
	if err != nil {
		t.Fatal(err)
	}

	db := dynamodb.New(sess)
	table := "journal"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := CreateJournalTable(ctx, db, table); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := deleteTable(db, table); err != nil {
			t.Fatal(err)
		}
	})

	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{
				DB:    db,
				Table: table,
			}
		},
	)
}

func deleteTable(db *dynamodb.DynamoDB, table string) error {
	if _, err := db.DeleteTable(
		&dynamodb.DeleteTableInput{
			TableName: aws.String(table),
		},
	); err != nil {
		if !awsx.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
			return err
		}
	}

	return nil
}