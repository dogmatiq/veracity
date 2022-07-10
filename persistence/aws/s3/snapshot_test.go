package s3_test

import (
	"errors"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dogmatiq/marshalkit"
	. "github.com/dogmatiq/veracity/persistence/aws/s3"
	"github.com/dogmatiq/veracity/persistence/internal/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("type AggregateSnapshotReader and AggregateSnapshotWriter", func() {
	persistencetest.DeclareSnapshotTests(
		func(m marshalkit.ValueMarshaler) persistencetest.SnapshotContext {
			accessKey := os.Getenv("DOGMATIQ_TEST_MINIO_ACCESS_KEY")
			if accessKey == "" {
				accessKey = "minio"
			}

			secretKey := os.Getenv("DOGMATIQ_TEST_MINIO_SECRET_KEY")
			if secretKey == "" {
				secretKey = "password"
			}

			endpoint := os.Getenv("DOGMATIQ_TEST_MINIO_ENDPOINT")
			if endpoint == "" {
				endpoint = "http://localhost:29000"
			}

			config := &aws.Config{
				Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
				Endpoint:         aws.String(endpoint),
				Region:           aws.String("us-east-1"),
				DisableSSL:       aws.Bool(true),
				S3ForcePathStyle: aws.Bool(true),
			}

			sess := session.New(config)
			client := s3.New(sess)
			bucket := "snapshots"

			err := deleteBucket(client, bucket)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			_, err = client.CreateBucket(&s3.CreateBucketInput{
				Bucket: aws.String(bucket),
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			return persistencetest.SnapshotContext{
				Reader: &AggregateSnapshotReader{
					Client:    client,
					Bucket:    bucket,
					Marshaler: m,
				},
				Writer: &AggregateSnapshotWriter{
					Client:    client,
					Bucket:    bucket,
					Marshaler: m,
				},
				ArchiveIsHardDelete: true,
				AfterEach: func() {
					err := deleteBucket(client, bucket)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				},
			}
		},
	)
})

func deleteBucket(client *s3.S3, bucket string) error {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	for {
		out, err := client.ListObjectsV2(in)
		if err != nil {
			var awsErr awserr.Error
			if errors.As(err, &awsErr) {
				if awsErr.Code() == s3.ErrCodeNoSuchBucket {
					return nil
				}
			}

			return err
		}

		for _, obj := range out.Contents {
			if _, err := client.DeleteObject(&s3.DeleteObjectInput{
				Bucket: in.Bucket,
				Key:    obj.Key,
			}); err != nil {
				return err
			}
		}

		if !*out.IsTruncated {
			break
		}

		in.ContinuationToken = out.ContinuationToken
	}

	if _, err := client.DeleteBucket(
		&s3.DeleteBucketInput{Bucket: in.Bucket},
	); err != nil {
		return err
	}

	return nil
}
