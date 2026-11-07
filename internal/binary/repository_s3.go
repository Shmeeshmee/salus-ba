package binary

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// The Store on an S3-compatible object store through aws-sdk-go-v2 (ADR 0010): path-style addressing, the
// SDK's checksums only where the API requires them, and an unsigned payload on each put, because the body is
// a stream (it cannot be hashed before it is sent) and AES-GCM already authenticates every byte of it
// (SPEC.md §2 "Encryption"). Only the binary id travels as the key; nothing else of the file reaches the store.
type s3Store struct {
	client   *s3.Client
	bucket   string
	pageSize int32
}

// NewS3Store returns the Store on the bucket of the store at endpoint (an absolute http(s) URL) in region,
// signed with the key pair, over hc. The listing reads pageSize keys a page (S3's maximum is 1000).
func NewS3Store(hc *http.Client, endpoint, region, bucket, accessKeyID, secretAccessKey string, pageSize int32) Store {
	creds := aws.Credentials{AccessKeyID: accessKeyID, SecretAccessKey: secretAccessKey}
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       region,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return creds, nil
		}),
		UsePathStyle:               true,
		HTTPClient:                 hc,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &s3Store{client: client, bucket: bucket, pageSize: pageSize}
}

// Put sends one PutObject of size bytes. The body is read once, so the put is not retried: the caller has
// the file, the store does not.
func (s *s3Store) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	// Only Read: the SDK sends a body that is also an io.Closer (the upload's pipe) without its length, which
	// the store refuses.
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: struct{ io.Reader }{r}, ContentLength: aws.Int64(size),
	}, s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware), func(o *s3.Options) {
		o.RetryMaxAttempts = 1
	})
	return failed("put", err)
}

// GetRange sends one GetObject with the range bytes=from-to.
func (s *s3Store) GetRange(ctx context.Context, key string, from, to int64) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
		Range: aws.String("bytes=" + strconv.FormatInt(from, 10) + "-" + strconv.FormatInt(to, 10)),
	})
	if err != nil {
		return nil, failed("get", err)
	}
	return out.Body, nil
}

// Delete sends one DeleteObject, which S3 answers with success for a missing key too.
func (s *s3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return failed("delete", err)
}

// ListOlderThan walks ListObjectsV2 page by page with its continuation token.
func (s *s3Store) ListOlderThan(ctx context.Context, before time.Time, fn func(keys []string) error) error {
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket), MaxKeys: aws.Int32(s.pageSize),
	})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return failed("list", err)
		}
		var keys []string
		for _, o := range page.Contents {
			if o.Key != nil && o.LastModified != nil && o.LastModified.Before(before) {
				keys = append(keys, *o.Key)
			}
		}
		if len(keys) == 0 {
			continue
		}
		if err := fn(keys); err != nil {
			return err
		}
	}
	return nil
}

// failed maps a missing key to ErrObjectNotFound and keeps only the S3 error code of any other answer from
// the store, so no vendor error type leaves this file.
func failed(op string, err error) error {
	var missing *types.NoSuchKey
	var api smithy.APIError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &missing):
		return ErrObjectNotFound
	case errors.As(err, &api):
		return fmt.Errorf("binary: s3 %s: %s", op, api.ErrorCode())
	}
	return fmt.Errorf("binary: s3 %s: %w", op, err)
}
