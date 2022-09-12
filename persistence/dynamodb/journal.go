package dynamodb

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/persistence/internal/awsx"
)

// JournalOpener is an implementation of journal.BinaryOpener that opens
// journals that store records in a DynamoDB table.
type JournalOpener struct {
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

	// DecoratePutItem is an optional function that is called before each
	// DynamoDB "PutItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutItem func(*dynamodb.PutItemInput) []request.Option
}

// Open returns the journal at the given path.
//
// The path uniquely identifies the journal. It must not be empty. Each element
// must be a non-empty UTF-8 string consisting solely of printable Unicode
// characters, excluding whitespace. A printable character is any character from
// the Letter, Mark, Number, Punctuation or Symbol categories.
func (o *JournalOpener) Open(ctx context.Context, path ...string) (journal.BinaryJournal, error) {
	key := keyFromJournalPath(path)

	j := &binaryJournal{
		DB:              o.DB,
		DecorateGetItem: o.DecorateGetItem,
		DecoratePutItem: o.DecoratePutItem,

		Key: dynamodb.AttributeValue{S: &key},
	}

	j.ReadRequest = dynamodb.GetItemInput{
		TableName: aws.String(o.Table),
		Key: map[string]*dynamodb.AttributeValue{
			journalKeyAttr:     &j.Key,
			journalVersionAttr: &j.Version,
		},
		ProjectionExpression: aws.String(journalRecordAttr),
	}

	j.WriteRequest = dynamodb.PutItemInput{
		TableName:           aws.String(o.Table),
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

	return j, nil
}

// binaryJournal is an implementation of journal.BinaryJournal that stores
// records in a DynamoDB table.
type binaryJournal struct {
	DB              *dynamodb.DynamoDB
	DecorateGetItem func(*dynamodb.GetItemInput) []request.Option
	DecoratePutItem func(*dynamodb.PutItemInput) []request.Option

	Key          dynamodb.AttributeValue
	Version      dynamodb.AttributeValue
	Record       dynamodb.AttributeValue
	ReadRequest  dynamodb.GetItemInput
	WriteRequest dynamodb.PutItemInput
}

func (j *binaryJournal) Read(ctx context.Context, ver uint64) ([]byte, bool, error) {
	j.Version.N = aws.String(strconv.FormatUint(ver, 10))

	out, err := awsx.Do(
		ctx,
		j.DB.GetItemWithContext,
		j.DecorateGetItem,
		&j.ReadRequest,
	)
	if out.Item == nil || err != nil {
		return nil, false, err
	}

	return out.Item[journalRecordAttr].B, true, nil
}

func (j *binaryJournal) ReadOldest(ctx context.Context) (uint64, []byte, bool, error) {
	rec, ok, err := j.Read(ctx, 0)
	return 0, rec, ok, err
}

func (j *binaryJournal) Write(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	j.Version.N = aws.String(strconv.FormatUint(ver, 10))
	j.Record.B = rec

	_, err := awsx.Do(
		ctx,
		j.DB.PutItemWithContext,
		j.DecoratePutItem,
		&j.WriteRequest,
	)

	if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
		return false, nil
	}

	return true, err
}

// Close closes the journal.
func (j *binaryJournal) Close() error {
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

const (
	journalKeyAttr     = "K"
	journalVersionAttr = "V"
	journalRecordAttr  = "R"
)

func keyFromJournalPath(path []string) string {
	if len(path) == 0 {
		panic("path must not be empty")
	}

	var w strings.Builder

	for _, elem := range path {
		if len(elem) == 0 {
			panic("path element must not be empty")
		}

		if w.Len() > 0 {
			w.WriteByte('/')
		}

		for _, r := range elem {
			if r == '/' || r == '\\' {
				w.WriteByte('\\')
			}

			w.WriteRune(r)
		}
	}

	return w.String()
}
