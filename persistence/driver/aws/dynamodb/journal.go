package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/veracity/persistence/driver/aws/internal/awsx"
	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalStore is an implementation of journal.Store that contains journals
// that persist records in a DynamoDB table.
type JournalStore struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// Table is the table name used for storage of journal records.
	Table string

	// DecorateGetItem is an optional function that is called before each
	// DynamoDB "GetItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateGetItem func(*dynamodb.GetItemInput) []request.Option

	// DecorateQuery is an optional function that is called before each DynamoDB
	// "Query" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateQuery func(*dynamodb.QueryInput) []request.Option

	// DecoratePutItem is an optional function that is called before each
	// DynamoDB "PutItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutItem func(*dynamodb.PutItemInput) []request.Option

	// DecorateDeleteItem is an optional function that is called before each
	// DynamoDB "DeleteItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []request.Option
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
		DB:                 s.DB,
		DecorateGetItem:    s.DecorateGetItem,
		DecorateQuery:      s.DecorateQuery,
		DecoratePutItem:    s.DecoratePutItem,
		DecorateDeleteItem: s.DecorateDeleteItem,

		Key: dynamodb.AttributeValue{S: &key},
	}

	j.GetRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]*dynamodb.AttributeValue{
			journalKeyAttr:     &j.Key,
			journalVersionAttr: &j.Version,
		},
		ProjectionExpression: aws.String(`#R`),
		ExpressionAttributeNames: map[string]*string{
			"#R": aws.String(journalRecordAttr),
		},
	}

	j.QueryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#K = :K AND #V >= :V`),
		ExpressionAttributeNames: map[string]*string{
			"#K": aws.String(journalKeyAttr),
			"#V": aws.String(journalVersionAttr),
			"#R": aws.String(journalRecordAttr),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":K": &j.Key,
			":V": &j.Version,
		},
		ProjectionExpression: aws.String("#V, #R"),
	}

	j.PutRequest = dynamodb.PutItemInput{
		TableName:           aws.String(s.Table),
		ConditionExpression: aws.String(`attribute_not_exists(#K)`),
		ExpressionAttributeNames: map[string]*string{
			"#K": aws.String(journalKeyAttr),
		},
		Item: map[string]*dynamodb.AttributeValue{
			journalKeyAttr:     &j.Key,
			journalVersionAttr: &j.Version,
			journalRecordAttr:  &j.Record,
		},
	}

	j.DeleteRequest = dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]*dynamodb.AttributeValue{
			journalKeyAttr:     &j.Key,
			journalVersionAttr: &j.Version,
		},
	}

	return j, nil
}

// journalHandle is an implementation of journal.Journal that stores records in
// a DynamoDB table.
type journalHandle struct {
	DB                 *dynamodb.DynamoDB
	DecorateGetItem    func(*dynamodb.GetItemInput) []request.Option
	DecorateQuery      func(*dynamodb.QueryInput) []request.Option
	DecoratePutItem    func(*dynamodb.PutItemInput) []request.Option
	DecorateDeleteItem func(*dynamodb.DeleteItemInput) []request.Option

	Key     dynamodb.AttributeValue
	Version dynamodb.AttributeValue
	Record  dynamodb.AttributeValue

	GetRequest    dynamodb.GetItemInput
	QueryRequest  dynamodb.QueryInput
	PutRequest    dynamodb.PutItemInput
	DeleteRequest dynamodb.DeleteItemInput
}

func (h *journalHandle) Get(ctx context.Context, ver uint64) ([]byte, bool, error) {
	h.Version.N = aws.String(strconv.FormatUint(ver, 10))

	out, err := awsx.Do(
		ctx,
		h.DB.GetItemWithContext,
		h.DecorateGetItem,
		&h.GetRequest,
	)
	if out.Item == nil || err != nil {
		return nil, false, err
	}

	return out.Item[journalRecordAttr].B, true, nil
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
	h.Version.N = aws.String(strconv.FormatUint(begin, 10))

	var expectVer uint64

	for {
		out, err := awsx.Do(
			ctx,
			h.DB.QueryWithContext,
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

			ver, err := strconv.ParseUint(aws.StringValue(attr.N), 10, 64)
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

			ok, err = fn(ctx, ver, attr.B)
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
	h.Version.N = aws.String(strconv.FormatUint(ver, 10))
	h.Record.B = rec

	_, err := awsx.Do(
		ctx,
		h.DB.PutItemWithContext,
		h.DecoratePutItem,
		&h.PutRequest,
	)

	if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
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

			h.Version.N = aws.String(strconv.FormatUint(v, 10))

			_, err := awsx.Do(
				ctx,
				h.DB.DeleteItemWithContext,
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
					AttributeName: aws.String(journalKeyAttr),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String(journalVersionAttr),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String(journalKeyAttr),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String(journalVersionAttr),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		},
	)

	if awsx.IsErrorCode(err, dynamodb.ErrCodeResourceInUseException) {
		return nil
	}

	return err
}
