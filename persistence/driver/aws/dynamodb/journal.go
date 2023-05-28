package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/dogmatiq/veracity/persistence/driver/aws/internal/awsx"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalStore is an implementation of [journal.Store] that persists journals
// in a DynamoDB table.
type JournalStore struct {
	// Client is the DynamoDB client to use.
	Client *dynamodb.Client

	// Table is the table name used for storage of journal records.
	Table string

	// DecorateGetItem is an optional function that is called before each
	// DynamoDB "GetItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateGetItem func(*dynamodb.GetItemInput) []func(*dynamodb.Options)

	// DecorateQuery is an optional function that is called before each DynamoDB
	// "Query" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateQuery func(*dynamodb.QueryInput) []func(*dynamodb.Options)

	// DecoratePutItem is an optional function that is called before each
	// DynamoDB "PutItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutItem func(*dynamodb.PutItemInput) []func(*dynamodb.Options)

	// DecorateDeleteItem is an optional function that is called before each
	// DynamoDB "DeleteItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []func(*dynamodb.Options)
}

const (
	journalNameAttr    = "Name"
	journalVersionAttr = "Version"
	journalRecordAttr  = "Record"
)

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	j := &journ{
		Client:             s.Client,
		DecorateGetItem:    s.DecorateGetItem,
		DecorateQuery:      s.DecorateQuery,
		DecoratePutItem:    s.DecoratePutItem,
		DecorateDeleteItem: s.DecorateDeleteItem,

		name: &types.AttributeValueMemberS{
			Value: name,
		},
		version: &types.AttributeValueMemberN{},
		record:  &types.AttributeValueMemberB{},
	}

	j.getRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalNameAttr:    j.name,
			journalVersionAttr: j.version,
		},
		ProjectionExpression: aws.String(`#R`),
		ExpressionAttributeNames: map[string]string{
			"#R": journalRecordAttr,
		},
	}

	j.queryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#N = :N AND #V >= :V`),
		ProjectionExpression:   aws.String("#V, #R"),
		ExpressionAttributeNames: map[string]string{
			"#N": journalNameAttr,
			"#V": journalVersionAttr,
			"#R": journalRecordAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":N": j.name,
			":V": j.version,
		},
	}

	j.putRequest = dynamodb.PutItemInput{
		TableName:           aws.String(s.Table),
		ConditionExpression: aws.String(`attribute_not_exists(#N)`),
		ExpressionAttributeNames: map[string]string{
			"#N": journalNameAttr,
		},
		Item: map[string]types.AttributeValue{
			journalNameAttr:    j.name,
			journalVersionAttr: j.version,
			journalRecordAttr:  j.record,
		},
	}

	j.deleteRequest = dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalNameAttr:    j.name,
			journalVersionAttr: j.version,
		},
	}

	return j, nil
}

// journ is an implementation of journal.Journal that stores records in
// a DynamoDB table.
type journ struct {
	Client             *dynamodb.Client
	DecorateGetItem    func(*dynamodb.GetItemInput) []func(*dynamodb.Options)
	DecorateQuery      func(*dynamodb.QueryInput) []func(*dynamodb.Options)
	DecoratePutItem    func(*dynamodb.PutItemInput) []func(*dynamodb.Options)
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []func(*dynamodb.Options)

	name    *types.AttributeValueMemberS
	version *types.AttributeValueMemberN
	record  *types.AttributeValueMemberB

	getRequest    dynamodb.GetItemInput
	queryRequest  dynamodb.QueryInput
	putRequest    dynamodb.PutItemInput
	deleteRequest dynamodb.DeleteItemInput
}

func (j *journ) Get(ctx context.Context, ver uint64) ([]byte, bool, error) {
	j.version.Value = strconv.FormatUint(ver, 10)

	out, err := awsx.Do(
		ctx,
		j.Client.GetItem,
		j.DecorateGetItem,
		&j.getRequest,
	)
	if out.Item == nil || err != nil {
		return nil, false, err
	}

	b := out.Item[journalRecordAttr].(*types.AttributeValueMemberB)

	return b.Value, true, nil
}

func (j *journ) Range(
	ctx context.Context,
	ver uint64,
	fn journal.RangeFunc,
) error {
	checkVersion := true

	return j.rangeQuery(
		ctx,
		ver,
		func(
			ctx context.Context,
			v uint64,
			rec []byte,
		) (bool, error) {
			if checkVersion {
				if v != ver {
					return false, errors.New("cannot range over truncated records")
				}
				checkVersion = false
			}

			return fn(ctx, v, rec)
		},
	)
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	return j.rangeQuery(ctx, 0, fn)
}

func (j *journ) rangeQuery(
	ctx context.Context,
	begin uint64,
	fn func(context.Context, uint64, []byte) (bool, error),
) error {
	j.queryRequest.ExclusiveStartKey = nil
	j.version.Value = strconv.FormatUint(begin, 10)

	var expectVer uint64

	for {
		out, err := awsx.Do(
			ctx,
			j.Client.Query,
			j.DecorateQuery,
			&j.queryRequest,
		)
		if err != nil {
			return err
		}

		for _, item := range out.Items {
			attr, ok := item[journalVersionAttr]
			if !ok {
				return errors.New("journal is corrupt: item is missing version attribute")
			}

			ver, err := strconv.ParseUint(attr.(*types.AttributeValueMemberN).Value, 10, 64)
			if err != nil {
				return err
			}

			if expectVer != 0 && ver != expectVer {
				return fmt.Errorf("journal is corrupt: item has incorrect version (%d), expected %d", ver, expectVer)
			}

			expectVer = ver + 1

			attr, ok = item[journalRecordAttr]
			if !ok {
				return errors.New("journal is corrupt: item is missing record attribute")
			}

			ok, err = fn(ctx, ver, attr.(*types.AttributeValueMemberB).Value)
			if !ok || err != nil {
				return err
			}
		}

		if out.LastEvaluatedKey == nil {
			return nil
		}

		j.queryRequest.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (j *journ) Append(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	j.version.Value = strconv.FormatUint(ver, 10)
	j.record.Value = rec

	_, err := awsx.Do(
		ctx,
		j.Client.PutItem,
		j.DecoratePutItem,
		&j.putRequest,
	)

	if errors.As(err, new(*types.ConditionalCheckFailedException)) {
		return false, nil
	}

	return true, err
}

func (j *journ) Truncate(ctx context.Context, ver uint64) error {
	return j.RangeAll(
		ctx,
		func(ctx context.Context, v uint64, _ []byte) (bool, error) {
			if v >= ver {
				return false, nil
			}

			j.version.Value = strconv.FormatUint(v, 10)

			_, err := awsx.Do(
				ctx,
				j.Client.DeleteItem,
				j.DecorateDeleteItem,
				&j.deleteRequest,
			)
			return true, err
		},
	)
}

func (j *journ) Close() error {
	return nil
}

// CreateJournalStoreTable creates a DynamoDB table for use with [JournalStore].
func CreateJournalStoreTable(
	ctx context.Context,
	client *dynamodb.Client,
	table string,
	decorators ...func(*dynamodb.CreateTableInput) []func(*dynamodb.Options),
) error {
	_, err := awsx.Do(
		ctx,
		client.CreateTable,
		func(in *dynamodb.CreateTableInput) []func(*dynamodb.Options) {
			var options []func(*dynamodb.Options)
			for _, dec := range decorators {
				options = append(options, dec(in)...)
			}

			return options
		},
		&dynamodb.CreateTableInput{
			TableName: aws.String(table),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String(journalNameAttr),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String(journalVersionAttr),
					AttributeType: types.ScalarAttributeTypeN,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(journalNameAttr),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String(journalVersionAttr),
					KeyType:       types.KeyTypeRange,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
		},
	)

	if errors.As(err, new(*types.ResourceInUseException)) {
		return nil
	}

	return err
}
