package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/persistence/aws/internal/awsx"
	"google.golang.org/protobuf/proto"
)

// DefaultAggregateEventTablePrefix is the default DynamoDB table to use for
// storing aggregate events.
const DefaultAggregateEventTablePrefix = "AggregateEvent."

// AggregateEventReader reads historical aggregate events from DynamoDB.
//
// It implements aggregate.EventReader.
type AggregateEventReader struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// TablePrefix is a prefix to use on all DynamoDB table names.
	//
	// If it is empty, DefaultAggregateEventTablePrefix is used instead.
	TablePrefix string

	// DecorateQuery is an optional function that is called before each DynamoDB
	// "Query" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateQuery func(*dynamodb.QueryInput) []request.Option
}

// ReadBounds returns the revisions that are the bounds of the relevant
// historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// When loading the instance, only those events from revisions in the half-open
// range [begin, end) should be applied to the aggregate root.
func (e *AggregateEventReader) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, end uint64, _ error) {
	out, err := awsx.Do(
		ctx,
		e.DB.QueryWithContext,
		e.DecorateQuery,
		&dynamodb.QueryInput{
			TableName: tableName(e.TablePrefix, hk),
			KeyConditionExpression: aws.String(
				`InstanceID = :id`,
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":id": {S: &id},
			},
			ProjectionExpression: aws.String(`Revision,BeginRevision`),
			ScanIndexForward:     aws.Bool(false),
			Limit:                aws.Int64(1),
		},
	)
	if err != nil {
		// Do not treat non-existent tables as an error, tables are created
		// on-demand for each aggregate handler.
		if !awsx.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
			return 0, 0, err
		}
	}

	if len(out.Items) == 0 {
		return 0, 0, nil
	}

	item := out.Items[0]

	begin, err = unmarshalRevision(item["BeginRevision"])
	if err != nil {
		return 0, 0, err
	}

	end, err = unmarshalRevision(item["Revision"])
	if err != nil {
		return 0, 0, err
	}

	return begin, end + 1, nil
}

// ReadEvents loads some historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// It returns an implementation-defined number of events sourced from revisions
// in the half-open range [begin, end).
//
// When begin == end there are no more historical events to read. Otherwise,
// call ReadEvents() again with begin = end to continue reading events.
//
// The behavior is undefined if begin is lower than the begin revision returned
// by ReadBounds(). Implementations should return an error in this case.
func (e *AggregateEventReader) ReadEvents(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (events []*envelopespec.Envelope, end uint64, _ error) {
	out, err := awsx.Do(
		ctx,
		e.DB.QueryWithContext,
		e.DecorateQuery,
		&dynamodb.QueryInput{
			TableName: tableName(e.TablePrefix, hk),
			KeyConditionExpression: aws.String(
				`InstanceID = :id AND Revision >= :begin`,
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":id":    {S: &id},
				":begin": marshalRevision(begin),
			},
			ProjectionExpression: aws.String(`Revision,EventEnvelopes`),
		},
	)
	if err != nil {
		// Do not treat non-existent tables as an error, tables are created
		// on-demand for each aggregate handler.
		if !awsx.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
			return nil, 0, err
		}
	}

	for _, item := range out.Items {
		rev, err := unmarshalRevision(item["Revision"])
		if err != nil {
			return nil, 0, err
		}

		if rev != begin {
			return nil, 0, fmt.Errorf(
				"revision %d is archived",
				begin,
			)
		}

		if attr, ok := item["EventEnvelopes"]; ok {
			envelopes, err := unmarshalEnvelopes(attr.L)
			if err != nil {
				return nil, 0, err
			}

			events = append(events, envelopes...)
		}

		begin++
	}

	return events, begin, nil
}

// AggregateEventWriter writes historical aggregate events to DynamoDB.
//
// It implements aggregate.EventWriter.
type AggregateEventWriter struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// TablePrefix is a prefix to use on all DynamoDB table names.
	//
	// If it is empty, DefaultAggregateEventTablePrefix is used instead.
	TablePrefix string

	// DecorateCreateTable is an optional function that is called before each
	// DynamoDB "CreateTable" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateCreateTable func(*dynamodb.CreateTableInput) []request.Option

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

	tables tableRegistry
}

// WriteEvents writes events that were recorded by an aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// begin sets the first revision for the instance such that in the future only
// events from revisions in the half-open range [begin, end + 1) are applied
// when loading the aggregate root. The behavior is undefined if begin is larger
// than end + 1.
//
// Events from revisions prior to begin are still made available to external
// event consumers, but will no longer be needed for loading aggregate roots and
// may be archived.
//
// end must be the current end revision, that is, the revision after the most
// recent revision of the instance. Otherwise, an "optimistic concurrency
// control" error occurs and no changes are persisted. The behavior is undefined
// if end is greater than the actual end revision.
//
// The events slice may be empty, which allows modifying the begin revision
// without recording any new events.
func (e *AggregateEventWriter) WriteEvents(
	ctx context.Context,
	hk, id string,
	begin, end uint64,
	events []*envelopespec.Envelope,
) error {
	envelopes, err := marshalEnvelopes(events)
	if err != nil {
		return err
	}

	table, err := e.createTable(ctx, hk)
	if err != nil {
		return err
	}

	item := map[string]*dynamodb.AttributeValue{
		"InstanceID":    {S: aws.String(id)},
		"Revision":      marshalRevision(end),
		"BeginRevision": marshalRevision(begin),
	}

	if len(envelopes) > 0 {
		item["EventEnvelopes"] = &dynamodb.AttributeValue{L: envelopes}
	}

	_, err = awsx.Do(
		ctx,
		e.DB.PutItemWithContext,
		e.DecoratePutItem,
		&dynamodb.PutItemInput{
			TableName: table,
			ConditionExpression: aws.String(
				`attribute_not_exists(InstanceID)`,
			),
			Item: item,
		},
	)
	if err != nil {
		if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
			return fmt.Errorf(
				"optimistic concurrency conflict, %d is not the next revision",
				end,
			)
		}

		return err
	}

	return nil
}

// createTable creates the DynamoDB table for the aggregate with the given key.
func (e *AggregateEventWriter) createTable(ctx context.Context, hk string) (*string, error) {
	return e.tables.Create(
		ctx,
		e.DB,
		hk,
		func() (*dynamodb.CreateTableInput, []request.Option) {
			in := &dynamodb.CreateTableInput{
				TableName: tableName(e.TablePrefix, hk),
				AttributeDefinitions: []*dynamodb.AttributeDefinition{
					{
						AttributeName: aws.String("InstanceID"),
						AttributeType: aws.String("S"),
					},
					{
						AttributeName: aws.String("Revision"),
						AttributeType: aws.String("N"),
					},
				},
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("InstanceID"),
						KeyType:       aws.String("HASH"),
					},
					{
						AttributeName: aws.String("Revision"),
						KeyType:       aws.String("RANGE"),
					},
				},
				BillingMode: aws.String("PAY_PER_REQUEST"),
				Tags: []*dynamodb.Tag{
					{
						Key:   aws.String("DogmaHandlerKey"),
						Value: aws.String(hk),
					},
				},
			}

			return in, awsx.Decorate(in, e.DecorateCreateTable)
		},
	)
}

// marshalRevision marshals a revision to a DynamoDB number.
func marshalRevision(rev uint64) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		N: aws.String(
			strconv.FormatUint(rev, 10),
		),
	}
}

// unmarshalRevision unmarshals a revision from a DynamoDB number.
func unmarshalRevision(attr *dynamodb.AttributeValue) (uint64, error) {
	if attr == nil {
		return 0, errors.New("cannot unmarshal empty revision attribute")
	}

	return strconv.ParseUint(
		aws.StringValue(attr.N),
		10,
		64,
	)
}

// marshalEnvelopes marshals the given envelopes into a DynamoDB list.
func marshalEnvelopes(envelopes []*envelopespec.Envelope) ([]*dynamodb.AttributeValue, error) {
	var attrs []*dynamodb.AttributeValue

	for _, envelope := range envelopes {
		data, err := proto.Marshal(envelope)
		if err != nil {
			return nil, err
		}

		attrs = append(attrs, &dynamodb.AttributeValue{B: data})
	}

	return attrs, nil
}

// unmarshalEnvelopes unmarshals the given DynamoDB list into envelopes.
func unmarshalEnvelopes(attrs []*dynamodb.AttributeValue) ([]*envelopespec.Envelope, error) {
	var envelopes []*envelopespec.Envelope

	for _, attr := range attrs {
		var env envelopespec.Envelope
		if err := proto.Unmarshal(attr.B, &env); err != nil {
			return nil, err
		}

		envelopes = append(envelopes, &env)
	}

	return envelopes, nil
}
