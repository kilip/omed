package s3x

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Driver interacts with an S3-compatible object storage service using AWS SDK v2.
type S3Driver struct {
	cfg     S3Config
	client  *s3.Client
	presign *s3.PresignClient
}

// NewS3Driver initializes a new S3Driver with the provided configuration.
func NewS3Driver(cfg S3Config) (*S3Driver, error) {
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = DefaultRegion
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.AccessKey != "" || cfg.SecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	} else {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(aws.AnonymousCredentials{}))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("s3x: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	return &S3Driver{
		cfg:     cfg,
		client:  client,
		presign: s3.NewPresignClient(client),
	}, nil
}

// EnsureBucket checks if the bucket exists and creates it if it does not.
func (d *S3Driver) EnsureBucket(ctx context.Context, bucket string) error {
	if bucket == "" {
		return fmt.Errorf("s3x: bucket name is required")
	}

	exists, err := d.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = d.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		var owned *types.BucketAlreadyOwnedByYou
		var alreadyExists *types.BucketAlreadyExists
		if errors.As(err, &owned) || errors.As(err, &alreadyExists) {
			return nil
		}
		return fmt.Errorf("s3x: create bucket %q: %w", bucket, err)
	}
	return nil
}

// BucketExists reports whether a bucket exists on the S3 server.
func (d *S3Driver) BucketExists(ctx context.Context, bucket string) (bool, error) {
	_, err := d.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if isS3NotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("s3x: head bucket %q: %w", bucket, err)
	}
	return true, nil
}

// DeleteBucket deletes a bucket. When force is true, all objects inside the bucket are deleted first.
func (d *S3Driver) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	exists, err := d.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if force {
		if err := d.emptyBucket(ctx, bucket); err != nil {
			return err
		}
	}

	_, err = d.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil
		}
		return fmt.Errorf("s3x: delete bucket %q: %w", bucket, err)
	}
	return nil
}

func (d *S3Driver) emptyBucket(ctx context.Context, bucket string) error {
	var continuationToken *string
	for {
		out, err := d.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			if isS3NotFound(err) {
				return nil
			}
			return fmt.Errorf("s3x: list objects in %q: %w", bucket, err)
		}

		if len(out.Contents) > 0 {
			objs := make([]types.ObjectIdentifier, 0, len(out.Contents))
			for _, item := range out.Contents {
				if item.Key != nil {
					objs = append(objs, types.ObjectIdentifier{Key: item.Key})
				}
			}
			if len(objs) > 0 {
				_, err := d.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
					Bucket: aws.String(bucket),
					Delete: &types.Delete{
						Objects: objs,
						Quiet:   aws.Bool(true),
					},
				})
				if err != nil {
					return fmt.Errorf("s3x: batch delete objects in %q: %w", bucket, err)
				}
			}
		}

		if out.IsTruncated == nil || !*out.IsTruncated || out.NextContinuationToken == nil {
			break
		}
		continuationToken = out.NextContinuationToken
	}
	return nil
}

// PutObject uploads an object to the specified bucket and key.
func (d *S3Driver) PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) (ObjectInfo, error) {
	if bucket == "" || key == "" {
		return ObjectInfo{}, fmt.Errorf("s3x: bucket and key are required")
	}
	if body == nil {
		return ObjectInfo{}, fmt.Errorf("s3x: object body is required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	out, err := d.client.PutObject(ctx, input)
	if err != nil {
		if isS3BucketNotFound(err) {
			return ObjectInfo{}, ErrBucketNotFound
		}
		return ObjectInfo{}, fmt.Errorf("s3x: put object %s/%s: %w", bucket, key, err)
	}

	etag := ""
	if out.ETag != nil {
		etag = strings.Trim(*out.ETag, `"`)
	}

	return ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         size,
		ContentType:  contentType,
		ETag:         etag,
		LastModified: time.Now().UTC(),
	}, nil
}

// GetObject downloads an object stream and its metadata from S3.
func (d *S3Driver) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	out, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3BucketNotFound(err) {
			return nil, ErrBucketNotFound
		}
		if isS3NotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("s3x: get object %s/%s: %w", bucket, key, err)
	}

	info := ObjectInfo{
		Bucket:      bucket,
		Key:         key,
		ContentType: "application/octet-stream",
	}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil && *out.ContentType != "" {
		info.ContentType = *out.ContentType
	}
	if out.ETag != nil {
		info.ETag = strings.Trim(*out.ETag, `"`)
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}

	return &Object{
		ObjectInfo: info,
		Body:       out.Body,
	}, nil
}

// StatObject fetches object metadata via HeadObject without downloading the body.
func (d *S3Driver) StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	out, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3BucketNotFound(err) {
			return ObjectInfo{}, ErrBucketNotFound
		}
		if isS3NotFound(err) {
			return ObjectInfo{}, ErrObjectNotFound
		}
		return ObjectInfo{}, fmt.Errorf("s3x: stat object %s/%s: %w", bucket, key, err)
	}

	info := ObjectInfo{
		Bucket:      bucket,
		Key:         key,
		ContentType: "application/octet-stream",
	}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	if out.ContentType != nil && *out.ContentType != "" {
		info.ContentType = *out.ContentType
	}
	if out.ETag != nil {
		info.ETag = strings.Trim(*out.ETag, `"`)
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}

	return info, nil
}

// DeleteObject removes an object from S3.
func (d *S3Driver) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil
		}
		return fmt.Errorf("s3x: delete object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// PresignGetObject generates a presigned GET URL for downloading the specified object.
func (d *S3Driver) PresignGetObject(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = DefaultPresignTTL
	}
	req, err := d.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("s3x: presign get object %s/%s: %w", bucket, key, err)
	}
	return req.URL, nil
}

// Close releases any resources held by S3Driver.
func (d *S3Driver) Close() error {
	return nil
}

func isS3BucketNotFound(err error) bool {
	var noSuchBucket *types.NoSuchBucket
	return errors.As(err, &noSuchBucket)
}

func isS3NotFound(err error) bool {
	var notFound *types.NotFound
	var noSuchBucket *types.NoSuchBucket
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &notFound) || errors.As(err, &noSuchBucket) || errors.As(err, &noSuchKey) {
		return true
	}
	var respErr *awshttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}
