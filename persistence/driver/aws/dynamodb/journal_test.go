package dynamodb_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	. "github.com/dogmatiq/veracity/persistence/driver/aws/dynamodb"
	"github.com/dogmatiq/veracity/persistence/journal"
)

func TestJournal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	endpoint := os.Getenv("DOGMATIQ_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:28000"
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...any) (aws.Endpoint, error) {
					return aws.Endpoint{URL: endpoint}, nil
				},
			),
		),
		config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "<id>",
					SecretAccessKey: "<secret>",
					SessionToken:    "",
				},
			},
		),
		config.WithRetryer(
			func() aws.Retryer {
				return aws.NopRetryer{}
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg)

	table := "journal"

	if err := CreateJournalTable(ctx, client, table); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := deleteTable(ctx, client, table); err != nil {
			t.Fatal(err)
		}

		cancel()
	})

	journal.RunTests(
		t,
		func(t *testing.T) journal.Store {
			return &JournalStore{
				Client: client,
				Table:  table,
			}
		},
	)
}

func deleteTable(ctx context.Context, client *dynamodb.Client, table string) error {
	if _, err := client.DeleteTable(
		ctx,
		&dynamodb.DeleteTableInput{
			TableName: aws.String(table),
		},
	); err != nil {
		if !errors.As(err, new(*types.ResourceNotFoundException)) {
			return err
		}
	}

	return nil
}
