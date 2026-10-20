package s3x

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	// ErrBucketNotFound is returned when a requested bucket does not exist.
	ErrBucketNotFound = errors.New("s3x: bucket not found")
	// ErrBucketNotEmpty is returned when attempting to delete a non-empty bucket without force.
	ErrBucketNotEmpty = errors.New("s3x: bucket not empty")
	// ErrObjectNotFound is returned when a requested object does not exist.
	ErrObjectNotFound = errors.New("s3x: object not found")
)

// ObjectInfo holds metadata about a stored object.
type ObjectInfo struct {
	Bucket       string
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// Object represents a retrieved object stream along with its metadata.
type Object struct {
	ObjectInfo
	Body io.ReadCloser
}

// Driver specifies the low-level contract for interacting with object storage backends.
type Driver interface {
	// EnsureBucket creates the bucket if it does not already exist.
	EnsureBucket(ctx context.Context, bucket string) error
	// BucketExists reports whether the specified bucket exists.
	BucketExists(ctx context.Context, bucket string) (bool, error)
	// DeleteBucket removes a bucket. When force is true, all objects in the bucket are deleted first.
	DeleteBucket(ctx context.Context, bucket string, force bool) error
	// PutObject uploads an object to the specified bucket and key.
	PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) (ObjectInfo, error)
	// GetObject retrieves an object stream and metadata from the specified bucket and key.
	GetObject(ctx context.Context, bucket, key string) (*Object, error)
	// StatObject retrieves metadata for an object without downloading its body.
	StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error)
	// DeleteObject deletes an object from the specified bucket and key.
	DeleteObject(ctx context.Context, bucket, key string) error
	// PresignGetObject generates a presigned HTTP GET URL for downloading the object.
	PresignGetObject(ctx context.Context, bucket, key string, expires time.Duration) (string, error)
	// Close releases any resources held by the driver.
	Close() error
}
