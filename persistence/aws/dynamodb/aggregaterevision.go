package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/aggregate"
	"github.com/dogmatiq/veracity/persistence/aws/internal/awsx"
	"google.golang.org/protobuf/proto"
)

// AggregateRevisionReader reads historical aggregate events from DynamoDB.
//
// It implements aggregate.EventReader.
type AggregateRevisionReader struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// Table is the table name used for storage of aggregate revisions.
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
}

// ReadBounds returns the revisions that are the bounds of the relevant
// historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
func (r *AggregateRevisionReader) ReadBounds(
	ctx context.Context,
	hk, id string,
) (aggregate.Bounds, error) {
	out, err := awsx.Do(
		ctx,
		r.DB.QueryWithContext,
		r.DecorateQuery,
		&dynamodb.QueryInput{
			TableName: &r.Table,
			KeyConditionExpression: aws.String(
				`HandlerKeyAndInstanceID = :id`,
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":id": marshalAggregateRevisionPartitionID(hk, id),
			},
			ProjectionExpression: aws.String(`Revision, BeginRevision, CausationID, Uncommitted`),
			ScanIndexForward:     aws.Bool(false),
			Limit:                aws.Int64(1),
		},
	)
	if err != nil {
		return aggregate.Bounds{}, err
	}

	if len(out.Items) == 0 {
		return aggregate.Bounds{}, nil
	}

	item := out.Items[0]
	var bounds aggregate.Bounds

	bounds.End, err = unmarshalRevision(item["Revision"])
	if err != nil {
		return aggregate.Bounds{}, err
	}

	bounds.Begin, err = unmarshalRevision(item["BeginRevision"])
	if err != nil {
		return aggregate.Bounds{}, err
	}

	bounds.End++

	if unmarshalBool(item["Uncommitted"]) {
		bounds.UncommittedRevisionCausationID = unmarshalString(item["CausationID"])
	}

	return bounds, nil
}

// ReadRevisions loads some historical revisions for a specific aggregate
// instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// It returns an implementation-defined number of revisions starting with the
// begin revision.
//
// When the revisions slice is empty there are no more revisions to read.
// Otherwise, begin should be incremented by len(revisions) and ReadEvents()
// called again to continue reading revisions.
//
// The behavior is undefined if begin is lower than the begin revision returned
// by ReadBounds(). Implementations should return an error in this case.
func (r *AggregateRevisionReader) ReadRevisions(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (revisions []aggregate.Revision, _ error) {
	out, err := awsx.Do(
		ctx,
		r.DB.QueryWithContext,
		r.DecorateQuery,
		&dynamodb.QueryInput{
			TableName: &r.Table,
			KeyConditionExpression: aws.String(
				`HandlerKeyAndInstanceID = :id AND Revision >= :begin`,
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":id":    marshalAggregateRevisionPartitionID(hk, id),
				":begin": marshalRevision(begin),
			},
			ProjectionExpression: aws.String(`Revision, BeginRevision, CausationID, Events`),
		},
	)
	if err != nil {
		return nil, err
	}

	for _, item := range out.Items {
		var rev aggregate.Revision

		rev.End, err = unmarshalRevision(item["Revision"])
		if err != nil {
			return nil, err
		}

		if rev.End != begin {
			return nil, fmt.Errorf(
				"revision %d is archived",
				begin,
			)
		}

		rev.Begin, err = unmarshalRevision(item["BeginRevision"])
		if err != nil {
			return nil, err
		}

		rev.CausationID = unmarshalString(item["CausationID"])

		rev.Events, err = unmarshalEnvelopes(item["Events"])
		if err != nil {
			return nil, err
		}

		revisions = append(revisions, rev)
		begin++
	}

	return revisions, nil
}

// AggregateRevisionWriter writes historical aggregate events to DynamoDB.
//
// It implements aggregate.EventWriter.
type AggregateRevisionWriter struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// Table is the table name used for storage of aggregate revisions.
	Table string

	// DecoratePutItem is an optional function that is called before each
	// DynamoDB "PutItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutItem func(*dynamodb.PutItemInput) []request.Option

	// DecorateUpdateItem is an optional function that is called before each
	// DynamoDB "UpdateItem" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateUpdateItem func(*dynamodb.UpdateItemInput) []request.Option
}

// PrepareRevision prepares a new revision of an aggregate instance to be
// committed.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// rev.Begin sets the first revision for the instance. In the future only events
// from revisions in the half-open range [rev.Begin, rev.End + 1) are applied
// when loading the aggregate root. The behavior is undefined if rev.Begin is
// larger than rev.End + 1.
//
// Events from revisions prior to rev.Begin are still made available to external
// event consumers, but will no longer be needed for loading aggregate roots and
// may be archived.
//
// rev.End must be the current end revision, that is, the revision after the
// most recent revision of the instance. Otherwise, an "optimistic concurrency
// control" error occurs and no changes are persisted. The behavior is undefined
// if rev.End is greater than the actual end revision.
//
// The behavior is undefined if the most recent revision is uncommitted.
func (w *AggregateRevisionWriter) PrepareRevision(
	ctx context.Context,
	hk, id string,
	rev aggregate.Revision,
) error {
	if rev.CausationID == "" {
		panic("command ID must not be empty")
	}

	item := map[string]*dynamodb.AttributeValue{
		"HandlerKeyAndInstanceID": marshalAggregateRevisionPartitionID(hk, id),
		"Revision":                marshalRevision(rev.End),
		"BeginRevision":           marshalRevision(rev.Begin),
		"CausationID":             marshalString(rev.CausationID),
		"Uncommitted":             marshalBool(true),
	}

	if len(rev.Events) > 0 {
		events, err := marshalEnvelopes(rev.Events)
		if err != nil {
			return err
		}

		item["Events"] = events
	}

	_, err := awsx.Do(
		ctx,
		w.DB.PutItemWithContext,
		w.DecoratePutItem,
		&dynamodb.PutItemInput{
			TableName: &w.Table,
			ConditionExpression: aws.String(
				`attribute_not_exists(HandlerKeyAndInstanceID)`,
			),
			Item: item,
		},
	)
	if err != nil {
		if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
			return fmt.Errorf(
				"optimistic concurrency conflict, %d is not the next revision",
				rev.End,
			)
		}

		return err
	}

	return nil
}

// CommitRevision commits a prepared revision.
//
// Committing a revision indicates that the command that produced it will
// not be retried.
//
// It returns an error if the revision does not exist or has already been
// committed.
//
// The behavior is undefined if rev is greater than the most recent
// revision.
func (w *AggregateRevisionWriter) CommitRevision(
	ctx context.Context,
	hk, id string,
	rev uint64,
) error {
	out, err := awsx.Do(
		ctx,
		w.DB.UpdateItemWithContext,
		w.DecorateUpdateItem,
		&dynamodb.UpdateItemInput{
			TableName: &w.Table,
			Key: map[string]*dynamodb.AttributeValue{
				"HandlerKeyAndInstanceID": marshalAggregateRevisionPartitionID(hk, id),
				"Revision":                marshalRevision(rev),
			},
			ConditionExpression: aws.String(
				`attribute_exists(HandlerKeyAndInstanceID)`,
			),
			UpdateExpression: aws.String(
				`SET Uncommitted = :uncommitted`,
			),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":uncommitted": marshalBool(false),
			},
			ReturnValues: aws.String("UPDATED_OLD"),
		},
	)
	if err != nil {
		if awsx.IsErrorCode(err, dynamodb.ErrCodeConditionalCheckFailedException) {
			return fmt.Errorf(
				"revision %d does not exist",
				rev,
			)
		}

		return err
	}

	if !unmarshalBool(out.Attributes["Uncommitted"]) {
		return fmt.Errorf(
			"revision %d is already committed",
			rev,
		)
	}

	return nil
}

// marshalAggregateRevisionPartitionID marshals an aggregate instance ID.
func marshalAggregateRevisionPartitionID(hk, id string) *dynamodb.AttributeValue {
	var w strings.Builder

	w.Grow(len(hk) + len(id) + 1)

	// Handler keys are not allowed to contain whitespace, so by using a regular
	// space as a character we do not need any kind of escaping to be able to
	// split the IDs, if that becomes necessary.
	w.WriteString(hk)
	w.WriteRune(' ')
	w.WriteString(id)

	return marshalString(w.String())
}

// CreateAggregateRevisionTable creates a DynamoDB for use with
// AggregateRevisionReader and AggregateRevisionWriter.
func CreateAggregateRevisionTable(
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
			TableName: &table,
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("HandlerKeyAndInstanceID"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("Revision"),
					AttributeType: aws.String("N"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("HandlerKeyAndInstanceID"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("Revision"),
					KeyType:       aws.String("RANGE"),
				},
			},
			BillingMode: aws.String("PAY_PER_REQUEST"),
		},
	)

	return err
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

// marshalString marshals a string to a DynamoDB string.
func marshalString(v string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{S: &v}
}

// unmarshalString unmarshals a stirng from a DynamoDB string.
func unmarshalString(attr *dynamodb.AttributeValue) string {
	if attr != nil {
		return aws.StringValue(attr.S)
	}

	return ""
}

// marshalBool marshals a boolean to a DynamoDB boolean.
func marshalBool(v bool) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{BOOL: &v}
}

// unmarshalBool unmarshals a boolean from a DynamoDB boolean.
func unmarshalBool(attr *dynamodb.AttributeValue) bool {
	return attr != nil && aws.BoolValue(attr.BOOL)
}

// marshalEnvelopes marshals the given envelopes into a DynamoDB list.
func marshalEnvelopes(envelopes []*envelopespec.Envelope) (*dynamodb.AttributeValue, error) {
	var attrs []*dynamodb.AttributeValue

	for _, envelope := range envelopes {
		data, err := proto.Marshal(envelope)
		if err != nil {
			return nil, err
		}

		attrs = append(attrs, &dynamodb.AttributeValue{B: data})
	}

	return &dynamodb.AttributeValue{L: attrs}, nil
}

// unmarshalEnvelopes unmarshals the given DynamoDB list into envelopes.
func unmarshalEnvelopes(attr *dynamodb.AttributeValue) ([]*envelopespec.Envelope, error) {
	if attr == nil {
		return nil, nil
	}

	var envelopes []*envelopespec.Envelope

	for _, attr := range attr.L {
		var env envelopespec.Envelope
		if err := proto.Unmarshal(attr.B, &env); err != nil {
			return nil, err
		}

		envelopes = append(envelopes, &env)
	}

	return envelopes, nil
}
