package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Journal struct {
	// DB is the DynamoDB client to use.
	DB *dynamodb.DynamoDB

	// Table is the table name used for storage of journal records.
	Table string
}

// Read returns the record that was written to produce the version v of the
// journal.
//
// If the version does not exist ok is false.
func (j *Journal) Read(ctx context.Context, v uint64) (r []byte, ok bool, err error) {
	return nil, false, nil
}

// Write appends a new record to the journal.
//
// v must be the current version of the journal.
//
// If v < current then the record is not persisted; ok is false indicating an
// optimistic concurrency conflict.
//
// If v > current then the behavior is undefined.
func (j *Journal) Write(ctx context.Context, v uint64, r []byte) (ok bool, err error) {
	return true, nil
}

// Close closes the journal.
func (j *Journal) Close() error {
	return nil
}
