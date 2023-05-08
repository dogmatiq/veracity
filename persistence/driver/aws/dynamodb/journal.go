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
	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalStore is an implementation of journal.Store that contains journals
// that persist records in a DynamoDB table.
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
	journalKeyAttr     = "Key"
	journalVersionAttr = "Version"
	journalRecordAttr  = "Record"
)

// Open returns the journal at the given path.
//
// The path uniquely identifies the journal. It must not be empty. Each element
// must be a non-empty UTF-8 string consisting solely of printable Unicode
// characters, excluding whitespace. A printable character is any character from
// the Letter, Mark, Number, Punctuation or Symbol categories.
func (s *JournalStore) Open(ctx context.Context, path ...string) (journal.Journal, error) {
	key := pathkey.New(path)

	j := &journalHandle{
		Client:             s.Client,
		DecorateGetItem:    s.DecorateGetItem,
		DecorateQuery:      s.DecorateQuery,
		DecoratePutItem:    s.DecoratePutItem,
		DecorateDeleteItem: s.DecorateDeleteItem,

		Key:     &types.AttributeValueMemberS{Value: key},
		Version: &types.AttributeValueMemberN{},
		Record:  &types.AttributeValueMemberB{},
	}

	j.GetRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalKeyAttr:     j.Key,
			journalVersionAttr: j.Version,
		},
		ProjectionExpression: aws.String(`#R`),
		ExpressionAttributeNames: map[string]string{
			"#R": journalRecordAttr,
		},
	}

	j.QueryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#K = :K AND #V >= :V`),
		ExpressionAttributeNames: map[string]string{
			"#K": journalKeyAttr,
			"#V": journalVersionAttr,
			"#R": journalRecordAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":K": j.Key,
			":V": j.Version,
		},
		ProjectionExpression: aws.String("#V, #R"),
	}

	j.PutRequest = dynamodb.PutItemInput{
		TableName:           aws.String(s.Table),
		ConditionExpression: aws.String(`attribute_not_exists(#K)`),
		ExpressionAttributeNames: map[string]string{
			"#K": journalKeyAttr,
		},
		Item: map[string]types.AttributeValue{
			journalKeyAttr:     j.Key,
			journalVersionAttr: j.Version,
			journalRecordAttr:  j.Record,
		},
	}

	j.DeleteRequest = dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalKeyAttr:     j.Key,
			journalVersionAttr: j.Version,
		},
	}

	return j, nil
}

// journalHandle is an implementation of journal.Journal that stores records in
// a DynamoDB table.
type journalHandle struct {
	Client             *dynamodb.Client
	DecorateGetItem    func(*dynamodb.GetItemInput) []func(*dynamodb.Options)
	DecorateQuery      func(*dynamodb.QueryInput) []func(*dynamodb.Options)
	DecoratePutItem    func(*dynamodb.PutItemInput) []func(*dynamodb.Options)
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []func(*dynamodb.Options)

	Key     *types.AttributeValueMemberS
	Version *types.AttributeValueMemberN
	Record  *types.AttributeValueMemberB

	GetRequest    dynamodb.GetItemInput
	QueryRequest  dynamodb.QueryInput
	PutRequest    dynamodb.PutItemInput
	DeleteRequest dynamodb.DeleteItemInput
}

func (h *journalHandle) Get(ctx context.Context, ver uint64) ([]byte, bool, error) {
	h.Version.Value = strconv.FormatUint(ver, 10)

	out, err := awsx.Do(
		ctx,
		h.Client.GetItem,
		h.DecorateGetItem,
		&h.GetRequest,
	)
	if out.Item == nil || err != nil {
		return nil, false, err
	}

	b := out.Item[journalRecordAttr].(*types.AttributeValueMemberB)

	return b.Value, true, nil
}

func (h *journalHandle) Range(
	ctx context.Context,
	ver uint64,
	fn func(context.Context, []byte) (bool, error),
) error {
	checkVersion := true

	return h.rangeQuery(
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

			return fn(ctx, rec)
		},
	)
}

func (h *journalHandle) RangeAll(
	ctx context.Context,
	fn func(context.Context, uint64, []byte) (bool, error),
) error {
	return h.rangeQuery(ctx, 0, fn)
}

func (h *journalHandle) rangeQuery(
	ctx context.Context,
	begin uint64,
	fn func(context.Context, uint64, []byte) (bool, error),
) error {
	h.QueryRequest.ExclusiveStartKey = nil
	h.Version.Value = strconv.FormatUint(begin, 10)

	var expectVer uint64

	for {
		out, err := awsx.Do(
			ctx,
			h.Client.Query,
			h.DecorateQuery,
			&h.QueryRequest,
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

		h.QueryRequest.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (h *journalHandle) Append(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	h.Version.Value = strconv.FormatUint(ver, 10)
	h.Record.Value = rec

	_, err := awsx.Do(
		ctx,
		h.Client.PutItem,
		h.DecoratePutItem,
		&h.PutRequest,
	)

	if errors.As(err, new(*types.ConditionalCheckFailedException)) {
		return false, nil
	}

	return true, err
}

func (h *journalHandle) Truncate(ctx context.Context, ver uint64) error {
	return h.RangeAll(
		ctx,
		func(ctx context.Context, v uint64, _ []byte) (bool, error) {
			if v >= ver {
				return false, nil
			}

			h.Version.Value = strconv.FormatUint(v, 10)

			_, err := awsx.Do(
				ctx,
				h.Client.DeleteItem,
				h.DecorateDeleteItem,
				&h.DeleteRequest,
			)
			return true, err
		},
	)
}

func (h *journalHandle) Close() error {
	return nil
}

// CreateJournalTable creates a DynamoDB for storing journal records.
func CreateJournalTable(
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
					AttributeName: aws.String(journalKeyAttr),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String(journalVersionAttr),
					AttributeType: types.ScalarAttributeTypeN,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(journalKeyAttr),
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
