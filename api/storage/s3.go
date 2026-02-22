package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 implements Storage backed by Amazon S3.
type S3 struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	forcePathS bool // use path-style addressing (MinIO, LocalStack)
}

// NewS3 creates a new S3-backed storage.
// bucket is the S3 bucket name; region is the AWS region (e.g. "us-east-1").
// endpoint is optional — set for S3-compatible stores like MinIO.
func NewS3(ctx context.Context, bucket, region, endpoint string) (*S3, error) {
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required when STORAGE_MODE=s3")
	}

	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var s3opts []func(*s3.Options)
	forcePathS := false
	if endpoint != "" {
		s3opts = append(s3opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
		forcePathS = true
	}

	client := s3.NewFromConfig(cfg, s3opts...)
	presigner := s3.NewPresignClient(client)

	return &S3{
		client:     client,
		presigner:  presigner,
		bucket:     bucket,
		forcePathS: forcePathS,
	}, nil
}

func (s *S3) GenerateUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String("application/dicom"),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign s3 upload: %w", err)
	}
	return req.URL, nil
}

func (s *S3) GenerateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign s3 download: %w", err)
	}
	return req.URL, nil
}

func (s *S3) Store(ctx context.Context, key string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String("application/dicom"),
	})
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}
	return nil
}

func (s *S3) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}
	return out.Body, nil
}

func (s *S3) List(ctx context.Context, prefix string) ([]string, error) {
	p := prefix + "/"
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(p),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 list: %w", err)
	}

	var keys []string
	for _, obj := range out.Contents {
		if obj.Key != nil && *obj.Key != "" {
			keys = append(keys, *obj.Key)
		}
	}
	return keys, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	return nil
}

func (s *S3) Move(ctx context.Context, srcKey, dstKey string) error {
	// S3 has no native move — copy then delete.
	copySource := s.bucket + "/" + srcKey
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return fmt.Errorf("s3 copy: %w", err)
	}

	return s.Delete(ctx, srcKey)
}

func (s *S3) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, fmt.Errorf("s3 head object: %w", err)
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

// KeyToPath returns empty for cloud storage (files are not on the local filesystem).
func (s *S3) KeyToPath(_ string) string {
	return ""
}
