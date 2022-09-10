package dynamodb_test

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/veracity/internal/awsx"
	. "github.com/dogmatiq/veracity/journal/dynamodb"
	"github.com/dogmatiq/veracity/journal/journaltest"
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

	sess := session.New(config)
	db := dynamodb.New(sess)
	table := "journal"

	if err := deleteTable(db, table); err != nil {
		t.Fatal(err)
	}

	journaltest.RunTests(t, func() journaltest.TestContext {
		j := &Journal{
			DB:    db,
			Table: table,
		}

		return journaltest.TestContext{
			Journal: j,
			Cleanup: func() error {
				if err := deleteTable(db, table); err != nil {
					t.Fatal(err)
				}

				return j.Close()
			},
		}
	})
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
