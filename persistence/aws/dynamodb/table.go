package dynamodb

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/veracity/persistence/aws/internal/awsx"
)

type TableRegistry struct {
	// DecorateCreateTable is an optional function that is called before each
	// DynamoDB "CreateTable" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateCreateTable func(*dynamodb.CreateTableInput) []request.Option

	// DecorateDescribeTable is an optional function that is called before each
	// DynamoDB "DescribeTable" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateDescribeTable func(*dynamodb.DescribeTableInput) []request.Option

	tables sync.Map
}

type tableEntry struct {
	Name     string
	Token    chan struct{}
	IsActive bool
}

func (r *TableRegistry) Create(
	ctx context.Context,
	db *dynamodb.DynamoDB,
	key string,
	def func() *dynamodb.CreateTableInput,
) (string, error) {
	e := r.entry(key)

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-e.Token:
	}

	if e.IsActive {
		return e.Name, nil
	}

	defer func() {
		if e.IsActive {
			close(e.Token)
		} else {
			e.Token <- struct{}{}
		}
	}()

	if e.Name == "" {
		in := def()

		out, err := awsx.Do(
			ctx,
			db.CreateTableWithContext,
			r.DecorateCreateTable,
			in,
		)
		if err != nil {
			if !awsx.IsErrorCode(err, dynamodb.ErrCodeResourceInUseException) {
				return "", err
			}
		}

		e.Name = aws.StringValue(in.TableName)

		if out.TableDescription != nil && aws.StringValue(out.TableDescription.TableStatus) == "ACTIVE" {
			e.IsActive = true
			return e.Name, nil
		}
	}

	return r.poll(ctx, db, e)
}

func (r *TableRegistry) entry(key string) *tableEntry {
	if v, ok := r.tables.Load(key); ok {
		return v.(*tableEntry)
	}

	e := &tableEntry{
		Token: make(chan struct{}, 1),
	}
	e.Token <- struct{}{}

	v, _ := r.tables.LoadOrStore(key, e)
	return v.(*tableEntry)
}

func (r *TableRegistry) poll(
	ctx context.Context,
	db *dynamodb.DynamoDB,
	e *tableEntry,
) (string, error) {
	for {
		out, err := awsx.Do(
			ctx,
			db.DescribeTableWithContext,
			r.DecorateDescribeTable,
			&dynamodb.DescribeTableInput{
				TableName: &e.Name,
			},
		)
		if err != nil {
			if !awsx.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
				return "", err
			}
		}

		if out.Table != nil && aws.StringValue(out.Table.TableStatus) == "ACTIVE" {
			e.IsActive = true
			return e.Name, nil
		}

		if err := linger.Sleep(ctx, 50*time.Millisecond); err != nil {
			return "", err
		}
	}
}
