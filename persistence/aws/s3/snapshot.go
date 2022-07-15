package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/persistence/aws/internal/awsx"
)

// AggregateSnapshotReader reads snapshots of aggregate roots from an S3 bucket.
//
// It implements aggregate.SnapshotReader.
type AggregateSnapshotReader struct {
	// Client is the S3 client used to read snapshots.
	Client *s3.S3

	// Bucket is the name of the S3 bucket in which snapshots are stored.
	Bucket string

	// Marshaler is used to unmarshal S3 objects into aggregate root instances.
	Marshaler marshalkit.ValueMarshaler

	// DecorateGetObject is an optional function that is called before each S3
	// "GetObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateGetObject func(*s3.GetObjectInput) []request.Option
}

// ReadSnapshot updates the contents of r to match the most recent snapshot that
// was taken at or after minRev.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// If ok is false, no compatible snapshot was found at or after minRev; root is
// guaranteed not to have been modified. Otherwise, rev is the revision of the
// aggregate instance when the snapshot was taken.
//
// A snapshot is considered compatible if it can assigned to the underlying type
// of r.
func (s *AggregateSnapshotReader) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minRev uint64,
) (rev uint64, ok bool, _ error) {
	out, err := awsx.Do(
		ctx,
		s.Client.GetObjectWithContext,
		s.DecorateGetObject,
		&s3.GetObjectInput{
			Bucket: &s.Bucket,
			Key:    snapshotKey(hk, id),
		},
	)
	if err != nil {
		if awsx.IsErrorCode(err, s3.ErrCodeNoSuchKey) {
			return 0, false, nil
		}

		return 0, false, err
	}
	defer out.Body.Close()

	if aws.StringValue(out.ContentType) == "" {
		return 0, false, errors.New(
			"S3 object has an empty content-type",
		)
	}

	revString := out.Metadata[snapshotRevisionMetaDataKey]
	if revString == nil {
		return 0, false, fmt.Errorf(
			"S3 object meta-data is missing the %s key",
			snapshotRevisionMetaDataKey,
		)
	}

	rev, err = strconv.ParseUint(*revString, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf(
			"S3 object meta-data has an invalid value for %s: %w",
			snapshotRevisionMetaDataKey,
			err,
		)
	}

	if rev < minRev {
		return 0, false, nil
	}

	body, err := io.ReadAll(out.Body)
	if err != nil {
		return 0, false, err
	}

	v, err := s.Marshaler.Unmarshal(marshalkit.Packet{
		MediaType: *out.ContentType,
		Data:      body,
	})
	if err != nil {
		return 0, false, err
	}

	src := reflect.ValueOf(v).Elem()
	dst := reflect.ValueOf(r).Elem()

	if !src.Type().AssignableTo(dst.Type()) {
		return 0, false, nil
	}

	dst.Set(src)

	return rev, true, nil
}

// AggregateSnapshotWriter writes snapshots of aggregate roots to an S3 bucket.
//
// It implements aggregate.SnapshotWriter.
type AggregateSnapshotWriter struct {
	// Client is the S3 client used to write snapshots.
	Client *s3.S3

	// Bucket is the name of the S3 bucket in which snapshots are stored.
	Bucket string

	// Marshaler is used to marshal aggregate root instances into S3 objects.
	Marshaler marshalkit.ValueMarshaler

	// DecoratePutObject is an optional function that is called before each S3
	// "PutObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePutObject func(*s3.PutObjectInput) []request.Option

	// DecorateDeleteObject is an optional function that is called before each S3
	// "DeleteObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateDeleteObject func(*s3.DeleteObjectInput) []request.Option
}

// WriteSnapshot saves a snapshot of a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// rev is the revision of the aggregate instance as represented by r.
func (s *AggregateSnapshotWriter) WriteSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	rev uint64,
) error {
	p, err := s.Marshaler.Marshal(r)
	if err != nil {
		return err
	}

	_, err = awsx.Do(
		ctx,
		s.Client.PutObjectWithContext,
		s.DecoratePutObject,
		&s3.PutObjectInput{
			Bucket:      &s.Bucket,
			Key:         snapshotKey(hk, id),
			ContentType: &p.MediaType,
			Body:        bytes.NewReader(p.Data),
			Metadata: map[string]*string{
				snapshotRevisionMetaDataKey: aws.String(
					strconv.FormatUint(rev, 10),
				),
			},
		},
	)
	return err
}

// ArchiveSnapshots archives any existing snapshots of a specific instance.
//
// The precise meaning of "archive" is implementation-defined. It is typical to
// hard-delete the snapshots as they no longer serve a purpose and will not be
// required in the future.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
func (s *AggregateSnapshotWriter) ArchiveSnapshots(
	ctx context.Context,
	hk, id string,
) error {
	_, err := awsx.Do(
		ctx,
		s.Client.DeleteObjectWithContext,
		s.DecorateDeleteObject,
		&s3.DeleteObjectInput{
			Bucket: &s.Bucket,
			Key:    snapshotKey(hk, id),
		},
	)
	return err
}

// snapshotKey is the S3 key used to store snapshots for the given
// handler/instance.
func snapshotKey(hk, id string) *string {
	return aws.String(hk + "/" + id)
}

// snapshotRevisionMetaDataKey is the S3 metadata key used to store the revision
// of the snapshots.
const snapshotRevisionMetaDataKey = "Dogma-Snapshot-Revision"
