package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/veracity/internal/awsx"
)

// CreateTable creates a DynamoDB for storing journal records.
func CreateTable(
	ctx context.Context,
	db *dynamodb.DynamoDB,
	table string,
	decorators ...func(*dynamodb.CreateTableInput) []request.Option,
) error {
	_, err := awsx.Do(
		ctx,
		db.CreateTableWithContext,
		func(in *dynamodb.CreateTableInput) []request.Option {
			var options []request.Option
			for _, dec := range decorators {
				options = append(options, dec(in)...)
			}
			return options
		},
		&dynamodb.CreateTableInput{
			TableName: aws.String(table),
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("JournalKey"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Version"),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("JournalKey"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("Version"),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		},
	)

	return err
}
