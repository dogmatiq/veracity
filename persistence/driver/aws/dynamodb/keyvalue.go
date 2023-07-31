package dynamodb

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/dogmatiq/veracity/persistence/driver/aws/internal/awsx"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// KeyValueStore is an implementation of [kv.Store] that persists keyspaces in a
// DynamoDB table.
type KeyValueStore struct {
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
	kvKeyspaceAttr = "Keyspace"
	kvKeyAttr      = "Key"
	kvValueAttr    = "Value"
)

// Open returns the keyspace with the given name.
func (s *KeyValueStore) Open(ctx context.Context, name string) (kv.Keyspace, error) {
	ks := &keyspace{
		Client:             s.Client,
		DecorateGetItem:    s.DecorateGetItem,
		DecorateQuery:      s.DecorateQuery,
		DecoratePutItem:    s.DecoratePutItem,
		DecorateDeleteItem: s.DecorateDeleteItem,

		name:  &types.AttributeValueMemberS{Value: name},
		key:   &types.AttributeValueMemberB{},
		value: &types.AttributeValueMemberB{},
	}

	ks.getRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			kvKeyspaceAttr: ks.name,
			kvKeyAttr:      ks.key,
		},
		ProjectionExpression: aws.String(`#V`),
		ExpressionAttributeNames: map[string]string{
			"#V": kvValueAttr,
		},
	}

	// Has() requests an unknown attribute to avoid fetching unnecessary data.
	ks.hasRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			kvKeyspaceAttr: ks.name,
			kvKeyAttr:      ks.key,
		},
		ProjectionExpression: aws.String(`NonExistent`),
	}

	ks.queryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#S = :S`),
		ProjectionExpression:   aws.String("#K, #V"),
		ExpressionAttributeNames: map[string]string{
			"#S": kvKeyspaceAttr,
			"#K": kvKeyAttr,
			"#V": kvValueAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":S": ks.name,
		},
	}

	ks.putRequest = dynamodb.PutItemInput{
		TableName: aws.String(s.Table),
		Item: map[string]types.AttributeValue{
			kvKeyspaceAttr: ks.name,
			kvKeyAttr:      ks.key,
			kvValueAttr:    ks.value,
		},
	}

	ks.deleteRequest = dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			kvKeyspaceAttr: ks.name,
			kvKeyAttr:      ks.key,
		},
	}

	return ks, nil
}

type keyspace struct {
	Client             *dynamodb.Client
	DecorateGetItem    func(*dynamodb.GetItemInput) []func(*dynamodb.Options)
	DecorateQuery      func(*dynamodb.QueryInput) []func(*dynamodb.Options)
	DecoratePutItem    func(*dynamodb.PutItemInput) []func(*dynamodb.Options)
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []func(*dynamodb.Options)

	name  *types.AttributeValueMemberS
	key   *types.AttributeValueMemberB
	value *types.AttributeValueMemberB

	getRequest    dynamodb.GetItemInput
	hasRequest    dynamodb.GetItemInput
	queryRequest  dynamodb.QueryInput
	putRequest    dynamodb.PutItemInput
	deleteRequest dynamodb.DeleteItemInput
}

func (ks *keyspace) Get(ctx context.Context, k []byte) ([]byte, error) {
	ks.key.Value = k

	out, err := awsx.Do(
		ctx,
		ks.Client.GetItem,
		ks.DecorateGetItem,
		&ks.getRequest,
	)
	if err != nil || out.Item == nil {
		return nil, err
	}

	v, err := getAttr[*types.AttributeValueMemberB](out.Item, kvValueAttr)
	if err != nil {
		return nil, err
	}

	return v.Value, nil

}

func (ks *keyspace) Has(ctx context.Context, k []byte) (bool, error) {
	ks.key.Value = k

	out, err := awsx.Do(
		ctx,
		ks.Client.GetItem,
		ks.DecorateGetItem,
		&ks.hasRequest,
	)
	if err != nil {
		return false, err
	}

	return out.Item != nil, nil
}

func (ks *keyspace) Set(ctx context.Context, k, v []byte) error {
	if v == nil {
		return ks.delete(ctx, k)
	}

	return ks.set(ctx, k, v)
}

func (ks *keyspace) set(ctx context.Context, k, v []byte) error {
	ks.key.Value = k
	ks.value.Value = v

	_, err := awsx.Do(
		ctx,
		ks.Client.PutItem,
		ks.DecoratePutItem,
		&ks.putRequest,
	)

	return err
}

func (ks *keyspace) delete(ctx context.Context, k []byte) error {
	ks.key.Value = k

	_, err := awsx.Do(
		ctx,
		ks.Client.DeleteItem,
		ks.DecorateDeleteItem,
		&ks.deleteRequest,
	)

	return err
}

func (ks *keyspace) Range(
	ctx context.Context,
	fn kv.RangeFunc,
) error {
	ks.queryRequest.ExclusiveStartKey = nil

	for {
		out, err := awsx.Do(
			ctx,
			ks.Client.Query,
			ks.DecorateQuery,
			&ks.queryRequest,
		)
		if err != nil {
			return err
		}

		for _, item := range out.Items {
			key, err := getAttr[*types.AttributeValueMemberB](item, kvKeyAttr)
			if err != nil {
				return err
			}

			value, err := getAttr[*types.AttributeValueMemberB](item, kvValueAttr)
			if err != nil {
				return err
			}

			ok, err := fn(ctx, key.Value, value.Value)
			if !ok || err != nil {
				return err
			}
		}

		if out.LastEvaluatedKey == nil {
			return nil
		}

		ks.queryRequest.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (ks *keyspace) Close() error {
	return nil
}

// CreateKeyValueStoreTable creates a DynamoDB table for use with
// [KeyValueStore].
func CreateKeyValueStoreTable(
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
					AttributeName: aws.String(kvKeyspaceAttr),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String(kvKeyAttr),
					AttributeType: types.ScalarAttributeTypeB,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(kvKeyspaceAttr),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String(kvKeyAttr),
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
