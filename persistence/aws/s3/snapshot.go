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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
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

	// DecorateGet is an optional function that is called before each S3
	// "GetObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateGet func(*s3.GetObjectInput) []request.Option
}

// ReadSnapshot updates the contents of r to match the most recent snapshot that
// was taken at or after minOffset.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// If ok is false, no compatible snapshot was found at or after minOffset; root
// is guaranteed not to have been modified. Otherwise, snapshotOffset is the
// offset of the most recent event applied to the root when the snapshot was
// taken.
//
// A snapshot is considered compatible if it can assigned to the underlying type
// of r.
func (s *AggregateSnapshotReader) ReadSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	minOffset uint64,
) (snapshotOffset uint64, ok bool, _ error) {
	in := &s3.GetObjectInput{}
	in.SetBucket(s.Bucket)
	in.SetKey(snapshotKey(hk, id))

	var options []request.Option
	if s.DecorateGet != nil {
		options = append(options, s.DecorateGet(in)...)
	}

	out, err := s.Client.GetObjectWithContext(ctx, in, options...)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == s3.ErrCodeNoSuchKey {
				return 0, false, nil
			}
		}

		return 0, false, err
	}
	defer out.Body.Close()

	if out.ContentType == nil || *out.ContentType == "" {
		return 0, false, errors.New(
			"S3 object has an empty content-type",
		)
	}

	offsetString := out.Metadata[snapshotOffsetMetaDataKey]
	if offsetString == nil {
		return 0, false, fmt.Errorf(
			"S3 object meta-data is missing the %s key",
			snapshotOffsetMetaDataKey,
		)
	}

	snapshotOffset, err = strconv.ParseUint(*offsetString, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf(
			"S3 object meta-data has an invalid value for %s: %w",
			snapshotOffsetMetaDataKey,
			err,
		)
	}

	if snapshotOffset < minOffset {
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

	return snapshotOffset, true, nil
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

	// DecoratePut is an optional function that is called before each S3
	// "PutObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecoratePut func(*s3.PutObjectInput) []request.Option

	// DecorateDelete is an optional function that is called before each S3
	// "DeleteObject" request.
	//
	// It may modify the API input in-place. It returns options that will be
	// applied to the request.
	DecorateDelete func(*s3.DeleteObjectInput) []request.Option
}

// WriteSnapshot saves a snapshot of a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the
// aggregate instance ID.
//
// snapshotOffset is the offset of the most recent event that has been
// applied to r.
func (s *AggregateSnapshotWriter) WriteSnapshot(
	ctx context.Context,
	hk, id string,
	r dogma.AggregateRoot,
	snapshotOffset uint64,
) error {
	p, err := s.Marshaler.Marshal(r)
	if err != nil {
		return err
	}

	in := &s3.PutObjectInput{}
	in.SetBucket(s.Bucket)
	in.SetKey(snapshotKey(hk, id))
	in.SetContentType(p.MediaType)
	in.SetBody(bytes.NewReader(p.Data))
	in.SetMetadata(
		map[string]*string{
			snapshotOffsetMetaDataKey: aws.String(
				strconv.FormatUint(snapshotOffset, 10),
			),
		},
	)

	var options []request.Option
	if s.DecoratePut != nil {
		options = append(options, s.DecoratePut(in)...)
	}

	_, err = s.Client.PutObjectWithContext(ctx, in, options...)
	return err
}

// ArchiveSnapshots archives any existing snapshots of a specific aggregate
// instance.
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
	in := &s3.DeleteObjectInput{}
	in.SetBucket(s.Bucket)
	in.SetKey(snapshotKey(hk, id))

	var options []request.Option
	if s.DecorateDelete != nil {
		options = append(options, s.DecorateDelete(in)...)
	}

	_, err := s.Client.DeleteObjectWithContext(ctx, in, options...)
	return err
}

// snapshotKey is the S3 key used to store snapshots for the given
// handler/instance.
func snapshotKey(hk, id string) string {
	return hk + "/" + id
}

// snapshotOffsetMetaDataKey is the S3 metadata key used to store the offset of the
// snapshots.
const snapshotOffsetMetaDataKey = "Dogma-Snapshot-Offset"
