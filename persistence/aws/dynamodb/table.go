package dynamodb

import (
	"context"
	"encoding/base64"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/dogmatiq/linger"
	"github.com/dogmatiq/veracity/persistence/aws/internal/awsx"
)

// tableRegistry is a collection of information about DynamoDB tables.
//
// It is used to determine if a table already exists, and create it if not.
type tableRegistry struct {
	tables sync.Map // map[string]*tableEntry
}

// tableEntry stores information about a single DynamoDB table.
type tableEntry struct {
	// IsActive is non-zero if the table is active.
	//
	// If it is zero the table is ACTIVE. Otherwise, IsActiveSem must be used to
	// wait for the table to become active.
	IsActive uint32 // atomic

	// IsActiveSem is a semaphore used to prevent multiple goroutines from
	// creating the table at the same time and to signal when the table is
	// confirmed to have an ACTIVE status.
	//
	// A successful read from the channel gives the calling goroutine permission
	// to proceed.
	//
	// If the read succeeds because the channel is closed, the table is ACTIVE.
	// Otherwise, the the table is not yet active and it is the channel reader's
	// responsibility to wait until it is before closing the channel.
	IsActiveSem chan struct{}

	// Name is the table name.
	Name string
}

// Create creates a DynamoDB table for some arbitrary key.
//
// If the table does not exist, def() is called to obtain the table definition.
//
// It blocks until the table status is ACTIVE.
func (r *tableRegistry) Create(
	ctx context.Context,
	db *dynamodb.DynamoDB,
	key string,
	def func() (*dynamodb.CreateTableInput, []request.Option),
) (_ *string, err error) {
	e := r.entry(key)

	if atomic.LoadUint32(&e.IsActive) != 0 {
		return &e.Name, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case _, pending := <-e.IsActiveSem:
		if !pending {
			return &e.Name, nil
		}
	}

	defer func() {
		if err == nil {
			atomic.StoreUint32(&e.IsActive, 1)
			close(e.IsActiveSem)
		} else {
			e.IsActiveSem <- struct{}{}
		}
	}()

	in, options := def()
	e.Name = aws.StringValue(in.TableName)

	out, err := db.CreateTableWithContext(ctx, in, options...)
	if err != nil {
		if awsx.IsErrorCode(err, dynamodb.ErrCodeResourceInUseException) {
			return r.waitForActiveStatus(ctx, db, e)
		}
	}

	if aws.StringValue(out.TableDescription.TableStatus) == "ACTIVE" {
		return &e.Name, nil
	}

	return r.waitForActiveStatus(ctx, db, e)
}

// entry returns the table entry for a specific key, creating an entry if
// necessary.
func (r *tableRegistry) entry(key string) *tableEntry {
	if v, ok := r.tables.Load(key); ok {
		return v.(*tableEntry)
	}

	e := &tableEntry{
		IsActiveSem: make(chan struct{}, 1),
	}

	v, loaded := r.tables.LoadOrStore(key, e)
	if !loaded {
		e.IsActiveSem <- struct{}{}
	}

	return v.(*tableEntry)
}

// waitForActiveStatus polls the database until the table is active.
func (r *tableRegistry) waitForActiveStatus(
	ctx context.Context,
	db *dynamodb.DynamoDB,
	e *tableEntry,
) (*string, error) {
	// CODE COVERAGE: This function tends to remain totally uncovered, as the
	// DynamoDB local image is fast enough so as to appear immediately
	// consistent.
	for {
		if err := linger.Sleep(ctx, 100*time.Millisecond); err != nil {
			return nil, err
		}

		out, err := db.DescribeTableWithContext(
			ctx,
			&dynamodb.DescribeTableInput{
				TableName: &e.Name,
			},
		)
		if err != nil {
			if awsx.IsErrorCode(err, dynamodb.ErrCodeResourceNotFoundException) {
				continue
			}

			return nil, err
		}

		if aws.StringValue(out.Table.TableStatus) == "ACTIVE" {
			return &e.Name, nil
		}
	}
}

// tableName returns the table name to use for a specific handler key.
//
// Handler keys (which may consist of any printable unicode characters) are
// encoded using the base76 URL alphabet, which is compatiable with DynamoDBs
// table naming restrictions.
func tableName(prefix, hk string) *string {
	if prefix == "" {
		prefix = DefaultAggregateEventTablePrefix
	}

	suffix := base64.RawURLEncoding.EncodeToString([]byte(hk))
	return aws.String(prefix + suffix)
}
