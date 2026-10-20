package s3x

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Storage provides high-level workspace bucket and object storage operations.
type Storage interface {
	// WorkspaceBucket returns the deterministic bucket name for a workspace ID.
	WorkspaceBucket(workspaceID int64) string
	// EnsureWorkspaceBucket ensures the bucket for the given workspace ID exists and returns its name.
	EnsureWorkspaceBucket(ctx context.Context, workspaceID int64) (string, error)
	// DeleteWorkspaceBucket deletes all objects and the bucket for the given workspace ID.
	DeleteWorkspaceBucket(ctx context.Context, workspaceID int64) error
	// EnsureBucket creates the bucket if it does not already exist.
	EnsureBucket(ctx context.Context, bucket string) error
	// BucketExists reports whether the specified bucket exists.
	BucketExists(ctx context.Context, bucket string) (bool, error)
	// DeleteBucket deletes the specified bucket (and its contents when force is true).
	DeleteBucket(ctx context.Context, bucket string, force bool) error
	// PutObject uploads an object to the specified bucket and key.
	PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) (ObjectInfo, error)
	// GetObject retrieves an object stream and metadata from the specified bucket and key.
	GetObject(ctx context.Context, bucket, key string) (*Object, error)
	// StatObject retrieves metadata for an object without downloading its body.
	StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error)
	// DeleteObject deletes an object from the specified bucket and key.
	DeleteObject(ctx context.Context, bucket, key string) error
	// PresignGetObject generates a presigned GET URL using the provided TTL (or default TTL if <= 0).
	PresignGetObject(ctx context.Context, bucket, key string, expires time.Duration) (string, error)
	// Driver returns the underlying transport driver.
	Driver() Driver
	// Close releases any underlying driver resources.
	Close() error
}

type defaultStorage struct {
	cfg    Config
	driver Driver
	logger *slog.Logger
}

// New creates and initializes a ready-to-use Storage instance based on Config.
func New(cfg Config) (Storage, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	var driver Driver
	var err error

	switch cfg.Driver {
	case "s3":
		driver, err = NewS3Driver(cfg.S3)
		if err != nil {
			return nil, fmt.Errorf("s3x: init s3 driver: %w", err)
		}
	default:
		driver = NewMemoryDriver()
	}

	return &defaultStorage{
		cfg:    cfg,
		driver: driver,
		logger: cfg.Logger,
	}, nil
}

// NewWithDriver constructs a Storage instance wrapping an explicit Driver (useful for testing).
func NewWithDriver(cfg Config, driver Driver) Storage {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &defaultStorage{
		cfg:    cfg,
		driver: driver,
		logger: cfg.Logger,
	}
}

func (s *defaultStorage) WorkspaceBucket(workspaceID int64) string {
	return s.cfg.WorkspaceBucket(workspaceID)
}

func (s *defaultStorage) EnsureWorkspaceBucket(ctx context.Context, workspaceID int64) (string, error) {
	if workspaceID <= 0 {
		return "", fmt.Errorf("s3x: invalid workspace id %d", workspaceID)
	}
	bucket := s.WorkspaceBucket(workspaceID)
	if err := s.driver.EnsureBucket(ctx, bucket); err != nil {
		return "", err
	}
	return bucket, nil
}

func (s *defaultStorage) DeleteWorkspaceBucket(ctx context.Context, workspaceID int64) error {
	if workspaceID <= 0 {
		return fmt.Errorf("s3x: invalid workspace id %d", workspaceID)
	}
	bucket := s.WorkspaceBucket(workspaceID)
	return s.driver.DeleteBucket(ctx, bucket, true)
}

func (s *defaultStorage) EnsureBucket(ctx context.Context, bucket string) error {
	return s.driver.EnsureBucket(ctx, bucket)
}

func (s *defaultStorage) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return s.driver.BucketExists(ctx, bucket)
}

func (s *defaultStorage) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	return s.driver.DeleteBucket(ctx, bucket, force)
}

func (s *defaultStorage) PutObject(ctx context.Context, bucket, key string, body io.Reader, size int64, contentType string) (ObjectInfo, error) {
	return s.driver.PutObject(ctx, bucket, key, body, size, contentType)
}

func (s *defaultStorage) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	return s.driver.GetObject(ctx, bucket, key)
}

func (s *defaultStorage) StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	return s.driver.StatObject(ctx, bucket, key)
}

func (s *defaultStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	return s.driver.DeleteObject(ctx, bucket, key)
}

func (s *defaultStorage) PresignGetObject(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = s.cfg.NormalizedPresignTTL()
	}
	return s.driver.PresignGetObject(ctx, bucket, key, expires)
}

func (s *defaultStorage) Driver() Driver {
	return s.driver
}

func (s *defaultStorage) Close() error {
	if s.driver != nil {
		return s.driver.Close()
	}
	return nil
}
