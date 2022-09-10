package dynamodb

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/veracity/internal/awsx"
)

type Journal struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// Table is the table name used for storage of journal records.
	Table string

	// Key uniquely identifies the journal.
	Key string

	// DecorateGetItem is an optional function that is called before each
	// DynamoDB "GetItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateGetItem func(*dynamodb.GetItemInput) []request.Option

	// DecoratePutItem is an optional function that is called before each
	// DynamoDB "PutItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutItem func(*dynamodb.PutItemInput) []request.Option
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *Journal) Read(ctx context.Context, v uint64) (r []byte, ok bool, err error) {
	out, err := awsx.Do(
		ctx,
		j.DB.GetItemWithContext,
		j.DecorateGetItem,
		&dynamodb.GetItemInput{
			TableName: aws.String(j.Table),
			Key: map[string]*dynamodb.AttributeValue{
				"K": marshalString(j.Key),
				"V": marshalVersion(v),
			},
			AttributesToGet: []*string{
				aws.String("R"),
			},
		},
	)
	if out.Item == nil || err != nil {
		return nil, false, err
	}

	return unmarshalRecord(out.Item["R"]), true, nil
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// If v > current then the behavior is undefined.
func (j *Journal) Write(ctx context.Context, v uint64, r []byte) (ok bool, err error) {
	if _, err = awsx.Do(
		ctx,
		j.DB.PutItemWithContext,
		j.DecoratePutItem,
		&dynamodb.PutItemInput{
			TableName: aws.String(j.Table),
			ConditionExpression: aws.String(
				`attribute_not_exists(K)`,
			),
			Item: map[string]*dynamodb.AttributeValue{
				"K": marshalString(j.Key),
				"V": marshalVersion(v),
				"R": marshalRecord(r),
			},
		},
	); err != nil {
		if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Close closes the journal.
func (j *Journal) Close() error {
	return nil
}

// marshalVersion marshals a version to a DynamoDB number.
func marshalVersion(v uint64) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		N: aws.String(
			strconv.FormatUint(v, 10),
		),
	}
}

// marshalString marshals a string to a DynamoDB string.
func marshalString(v string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{S: &v}
}

// marshalRecord marshals a record to a DynamoDB binary value.
func marshalRecord(r []byte) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{B: r}
}

// unmarshalRecord unmarshals a record from a DynamoDB binary value.
func unmarshalRecord(v *dynamodb.AttributeValue) []byte {
	return v.B
}
