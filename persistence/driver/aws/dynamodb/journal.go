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
	journalNameAttr     = "Name"
	journalPositionAttr = "Position"
	journalRecordAttr   = "Record"
)

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	j := &journ{
		Client:             s.Client,
		DecorateGetItem:    s.DecorateGetItem,
		DecorateQuery:      s.DecorateQuery,
		DecoratePutItem:    s.DecoratePutItem,
		DecorateDeleteItem: s.DecorateDeleteItem,

		name:     &types.AttributeValueMemberS{Value: name},
		position: &types.AttributeValueMemberN{},
		record:   &types.AttributeValueMemberB{},
	}

	j.boundsQueryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#N = :N`),
		ProjectionExpression:   aws.String("#P"),
		ExpressionAttributeNames: map[string]string{
			"#N": journalNameAttr,
			"#P": journalPositionAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":N": j.name,
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int32(1),
	}

	j.getRequest = dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalNameAttr:     j.name,
			journalPositionAttr: j.position,
		},
		ProjectionExpression: aws.String(`#R`),
		ExpressionAttributeNames: map[string]string{
			"#R": journalRecordAttr,
		},
	}

	j.rangeQueryRequest = dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String(`#N = :N AND #P >= :P`),
		ProjectionExpression:   aws.String("#P, #R"),
		ExpressionAttributeNames: map[string]string{
			"#N": journalNameAttr,
			"#P": journalPositionAttr,
			"#R": journalRecordAttr,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":N": j.name,
			":P": j.position,
		},
	}

	j.putRequest = dynamodb.PutItemInput{
		TableName:           aws.String(s.Table),
		ConditionExpression: aws.String(`attribute_not_exists(#N)`),
		ExpressionAttributeNames: map[string]string{
			"#N": journalNameAttr,
		},
		Item: map[string]types.AttributeValue{
			journalNameAttr:     j.name,
			journalPositionAttr: j.position,
			journalRecordAttr:   j.record,
		},
	}

	j.deleteRequest = dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]types.AttributeValue{
			journalNameAttr:     j.name,
			journalPositionAttr: j.position,
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

	name     *types.AttributeValueMemberS
	position *types.AttributeValueMemberN
	record   *types.AttributeValueMemberB

	boundsQueryRequest dynamodb.QueryInput
	getRequest         dynamodb.GetItemInput
	rangeQueryRequest  dynamodb.QueryInput
	putRequest         dynamodb.PutItemInput
	deleteRequest      dynamodb.DeleteItemInput
}

func (j *journ) Bounds(ctx context.Context) (begin, end journal.Position, err error) {
	*j.boundsQueryRequest.ScanIndexForward = true
	out, err := awsx.Do(
		ctx,
		j.Client.Query,
		j.DecorateQuery,
		&j.boundsQueryRequest,
	)
	if err != nil || len(out.Items) == 0 {
		return 0, 0, err
	}

	begin, err = parsePosition(out.Items[0])
	if err != nil {
		return 0, 0, err
	}

	*j.boundsQueryRequest.ScanIndexForward = false
	out, err = awsx.Do(
		ctx,
		j.Client.Query,
		j.DecorateQuery,
		&j.boundsQueryRequest,
	)
	if err != nil || len(out.Items) == 0 {
		return 0, 0, err
	}

	end, err = parsePosition(out.Items[0])
	if err != nil {
		return 0, 0, err
	}

	return begin, end + 1, nil
}

func (j *journ) Get(ctx context.Context, pos journal.Position) ([]byte, error) {
	j.position.Value = formatPosition(pos)

	out, err := awsx.Do(
		ctx,
		j.Client.GetItem,
		j.DecorateGetItem,
		&j.getRequest,
	)
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, journal.ErrNotFound
	}

	rec, err := getAttr[*types.AttributeValueMemberB](out.Item, journalRecordAttr)
	if err != nil {
		return nil, err
	}

	return rec.Value, nil
}

func (j *journ) Range(
	ctx context.Context,
	begin journal.Position,
	fn journal.RangeFunc,
) error {
	j.rangeQueryRequest.ExclusiveStartKey = nil
	j.position.Value = formatPosition(begin)

	expectPos := begin

	for {
		out, err := awsx.Do(
			ctx,
			j.Client.Query,
			j.DecorateQuery,
			&j.rangeQueryRequest,
		)
		if err != nil {
			return err
		}

		for _, item := range out.Items {
			pos, err := parsePosition(item)
			if err != nil {
				return err
			}

			if pos != expectPos {
				return journal.ErrNotFound
			}

			expectPos++

			rec, err := getAttr[*types.AttributeValueMemberB](item, journalRecordAttr)
			if err != nil {
				return err
			}

			ok, err := fn(ctx, pos, rec.Value)
			if !ok || err != nil {
				return err
			}
		}

		if out.LastEvaluatedKey == nil {
			return nil
		}

		j.rangeQueryRequest.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (j *journ) Append(ctx context.Context, end journal.Position, rec []byte) error {
	j.position.Value = formatPosition(end)
	j.record.Value = rec

	_, err := awsx.Do(
		ctx,
		j.Client.PutItem,
		j.DecoratePutItem,
		&j.putRequest,
	)

	if errors.As(err, new(*types.ConditionalCheckFailedException)) {
		return journal.ErrConflict
	}

	return err
}

func (j *journ) Truncate(ctx context.Context, end journal.Position) error {
	begin, actualEnd, err := j.Bounds(ctx)
	if err != nil {
		return err
	}

	if end >= actualEnd {
		return errors.New("cannot truncate beyond the end of the journal")
	}

	for pos := begin; pos < end; pos++ {
		j.position.Value = formatPosition(pos)

		if _, err := awsx.Do(
			ctx,
			j.Client.DeleteItem,
			j.DecorateDeleteItem,
			&j.deleteRequest,
		); err != nil {
			return err
		}
	}

	return nil
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
					AttributeName: aws.String(journalPositionAttr),
					AttributeType: types.ScalarAttributeTypeN,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String(journalNameAttr),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String(journalPositionAttr),
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

// parsePosition parses the position attribute in the given item.
func parsePosition(item map[string]types.AttributeValue) (journal.Position, error) {
	attr, err := getAttr[*types.AttributeValueMemberN](item, journalPositionAttr)
	if err != nil {
		return 0, err
	}

	pos, err := strconv.ParseUint(attr.Value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("item is corrupt: invalid position: %w", err)
	}

	return journal.Position(pos), nil
}

func formatPosition(pos journal.Position) string {
	return strconv.FormatUint(uint64(pos), 10)
}
