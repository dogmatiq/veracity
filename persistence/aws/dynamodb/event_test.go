package dynamodb_test

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	. "github.com/dogmatiq/veracity/persistence/aws/dynamodb"
	"github.com/dogmatiq/veracity/persistence/internal/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("type AggregateEventReader and AggregateEventWriter", func() {
	persistencetest.DeclareAggregateEventTests(
		func() persistencetest.AggregateEventContext {
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
			table := "events"

			err := deleteAllTables(db, table)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			return persistencetest.AggregateEventContext{
				Reader: &AggregateEventReader{
					DB: db,
				},
				Writer: &AggregateEventWriter{
					DB: db,
				},
				CanReadRevisionsBeforeBegin: true,
				AfterEach: func() {
					err := deleteAllTables(db, table)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				},
			}
		},
	)
})

func deleteAllTables(db *dynamodb.DynamoDB, table string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	in := &dynamodb.ListTablesInput{}

	for {
		out, err := db.ListTablesWithContext(ctx, in)
		if err != nil {
			return err
		}
		if len(out.TableNames) == 0 {
			return nil
		}

		for _, name := range out.TableNames {
			if _, err := db.DeleteTable(&dynamodb.DeleteTableInput{
				TableName: name,
			}); err != nil {
				return err
			}
		}

		if out.LastEvaluatedTableName == nil {
			return nil
		}

		in.ExclusiveStartTableName = out.LastEvaluatedTableName
	}
}
